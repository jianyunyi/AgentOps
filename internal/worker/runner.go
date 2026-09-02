package worker

import (
	"context"
	"errors"
	"log"
	"time"

	"agentscope/internal/outbox"
	platformmetrics "agentscope/internal/platform/metrics"
	redisv9 "github.com/redis/go-redis/v9"
)

type RunOptions struct {
	BatchSize    int
	Block        time.Duration
	ErrorBackoff func(attempt int) time.Duration
	Sleep        func(context.Context, time.Duration) error
	Logger       func(format string, args ...any)
	Metrics      *platformmetrics.WorkerMetrics
}

func defaultRunOptions() RunOptions {
	return RunOptions{
		BatchSize:    10,
		Block:        5 * time.Second,
		ErrorBackoff: RuntimeErrorBackoff,
		Sleep:        sleepContext,
		Logger:       log.Printf,
	}
}

func normalizeRunOptions(options RunOptions) RunOptions {
	defaults := defaultRunOptions()
	if options.BatchSize <= 0 {
		options.BatchSize = defaults.BatchSize
	}
	if options.Block <= 0 {
		options.Block = defaults.Block
	}
	if options.ErrorBackoff == nil {
		options.ErrorBackoff = defaults.ErrorBackoff
	}
	if options.Sleep == nil {
		options.Sleep = defaults.Sleep
	}
	if options.Logger == nil {
		options.Logger = defaults.Logger
	}
	return options
}

func RunOutbox(ctx context.Context, publisher *outbox.Publisher, interval time.Duration) error {
	return RunOutboxPublisherWithOptions(ctx, publisher, interval, RunOptions{})
}

func RunOutboxWithOptions(ctx context.Context, publish func(context.Context) error, interval time.Duration, options RunOptions) error {
	return runOutbox(ctx, func(ctx context.Context) (outbox.PublishOutcome, error) {
		err := publish(ctx)
		if err != nil {
			return outbox.PublishOutcome{Status: outbox.PublishStatusFailed}, err
		}
		return outbox.PublishOutcome{Status: outbox.PublishStatusDelivered}, nil
	}, interval, options)
}

func RunOutboxPublisherWithOptions(ctx context.Context, publisher *outbox.Publisher, interval time.Duration, options RunOptions) error {
	return runOutbox(ctx, publisher.PublishOneOutcome, interval, options)
}

func runOutbox(ctx context.Context, publish func(context.Context) (outbox.PublishOutcome, error), interval time.Duration, options RunOptions) error {
	options = normalizeRunOptions(options)
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	consecutiveErrors := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		outcome, err := publish(ctx)
		observeOutboxOutcome(options, outcome)
		failureErr := err
		if failureErr == nil && outcome.Status == outbox.PublishStatusFailed {
			failureErr = errors.New("outbox publish attempt failed")
		}
		if failureErr != nil {
			if cancellation := stopError(ctx, failureErr); cancellation != nil {
				return cancellation
			}
			consecutiveErrors++
			backoff := options.ErrorBackoff(consecutiveErrors)
			options.Logger("component=outbox_worker event=publish_error attempt=%d backoff=%s error=%q", consecutiveErrors, backoff, failureErr.Error())
			if err := options.Sleep(ctx, backoff); err != nil {
				return err
			}
			continue
		}
		consecutiveErrors = 0
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func Run(ctx context.Context, consumer *StreamConsumer, processor *AnalysisProcessor) error {
	return RunWithOptions(ctx, consumer, processor, RunOptions{})
}

func RunWithOptions(ctx context.Context, consumer *StreamConsumer, processor *AnalysisProcessor, options RunOptions) error {
	options = normalizeRunOptions(options)
	if err := consumer.EnsureGroup(ctx); err != nil {
		return err
	}
	consecutiveErrors := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pending, err := consumer.ClaimPending(ctx, options.BatchSize)
		if err != nil {
			if cancellation := stopError(ctx, err); cancellation != nil {
				return cancellation
			}
			consecutiveErrors++
			backoff := options.ErrorBackoff(consecutiveErrors)
			options.Logger("component=analysis_worker event=pending_recovery_error stream=%s consumer=%s attempt=%d backoff=%s error=%q", AnalysisStream, consumer.ConsumerID(), consecutiveErrors, backoff, err.Error())
			if err := options.Sleep(ctx, backoff); err != nil {
				return err
			}
			continue
		}
		consecutiveErrors = 0
		if options.Metrics != nil {
			options.Metrics.ObserveRecovered(len(pending))
		}
		if err := processMessages(ctx, consumer, processor, pending, options); err != nil {
			return err
		}

		batches, err := consumer.Read(ctx, options.BatchSize, options.Block)
		if err == redisv9.Nil {
			continue
		}
		if err != nil {
			if cancellation := stopError(ctx, err); cancellation != nil {
				return cancellation
			}
			observeRedisError(options, "read")
			consecutiveErrors++
			backoff := options.ErrorBackoff(consecutiveErrors)
			options.Logger("component=analysis_worker event=read_error stream=%s consumer=%s attempt=%d backoff=%s error=%q", AnalysisStream, consumer.ConsumerID(), consecutiveErrors, backoff, err.Error())
			if err := options.Sleep(ctx, backoff); err != nil {
				return err
			}
			continue
		}
		consecutiveErrors = 0
		for _, batch := range batches {
			if err := processMessages(ctx, consumer, processor, batch.Messages, options); err != nil {
				return err
			}
		}
	}
}

func processMessages(ctx context.Context, consumer *StreamConsumer, processor *AnalysisProcessor, messages []redisv9.XMessage, options RunOptions) error {
	for _, item := range messages {
		if err := processMessage(ctx, consumer, processor, item, options); err != nil {
			return err
		}
	}
	return nil
}

func processMessage(ctx context.Context, consumer *StreamConsumer, processor *AnalysisProcessor, item redisv9.XMessage, options RunOptions) error {
	message, err := DecodeAnalysisMessage(item.Values)
	if err != nil {
		attempt := AttemptFromValues(item.Values)
		if deadErr := consumer.DeadLetter(ctx, AnalysisMessage{EventID: item.ID}, attempt); deadErr != nil {
			observeRedisError(options, "dead_letter")
			logStreamFailure(options, "dead_letter_error", consumer, item.ID, attempt, deadErr)
			return stopError(ctx, deadErr)
		}
		if ackErr := consumer.Ack(ctx, item.ID); ackErr != nil {
			observeRedisError(options, "ack")
			logStreamFailure(options, "ack_error", consumer, item.ID, attempt, ackErr)
			return stopError(ctx, ackErr)
		}
		if options.Metrics != nil {
			options.Metrics.ObserveDeadLettered()
			options.Metrics.ObserveAck()
		}
		options.Logger("component=analysis_worker event=dead_letter stream=%s message_id=%s consumer=%s attempt=%d reason=decode_error", AnalysisStream, item.ID, consumer.ConsumerID(), attempt)
		return nil
	}

	attempt := AttemptFromValues(item.Values)
	decision, analysisErr := processor.ProcessWithError(ctx, message, attempt)
	if analysisErr != nil && options.Metrics != nil {
		options.Metrics.ObserveProcessingError()
	}
	switch decision {
	case DecisionAck:
		if err := consumer.Ack(ctx, item.ID); err != nil {
			observeRedisError(options, "ack")
			logStreamFailure(options, "ack_error", consumer, item.ID, attempt, err)
			return stopError(ctx, err)
		}
		if options.Metrics != nil {
			options.Metrics.ObserveAck()
		}
	case DecisionRetry:
		if err := options.Sleep(ctx, RetryDelay(attempt)); err != nil {
			return err
		}
		if err := consumer.Requeue(ctx, message, attempt+1); err != nil {
			observeRedisError(options, "requeue")
			logStreamFailure(options, "requeue_error", consumer, item.ID, attempt, err)
			return stopError(ctx, err)
		}
		if options.Metrics != nil {
			options.Metrics.ObserveRetried()
		}
		if err := consumer.Ack(ctx, item.ID); err != nil {
			observeRedisError(options, "ack")
			logStreamFailure(options, "ack_error", consumer, item.ID, attempt, err)
			return stopError(ctx, err)
		}
		options.Logger("component=analysis_worker event=retry stream=%s message_id=%s consumer=%s attempt=%d next_attempt=%d", AnalysisStream, item.ID, consumer.ConsumerID(), attempt, attempt+1)
	case DecisionDeadLetter:
		if err := consumer.DeadLetter(ctx, message, attempt); err != nil {
			observeRedisError(options, "dead_letter")
			logStreamFailure(options, "dead_letter_error", consumer, item.ID, attempt, err)
			return stopError(ctx, err)
		}
		if err := consumer.Ack(ctx, item.ID); err != nil {
			observeRedisError(options, "ack")
			logStreamFailure(options, "ack_error", consumer, item.ID, attempt, err)
			return stopError(ctx, err)
		}
		if options.Metrics != nil {
			options.Metrics.ObserveDeadLettered()
			options.Metrics.ObserveAck()
		}
		options.Logger("component=analysis_worker event=dead_letter stream=%s message_id=%s consumer=%s attempt=%d reason=processor", AnalysisStream, item.ID, consumer.ConsumerID(), attempt)
	}
	return nil
}

func stopError(ctx context.Context, err error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

func logStreamFailure(options RunOptions, event string, consumer *StreamConsumer, messageID string, attempt int, err error) {
	options.Logger("component=analysis_worker event=%s stream=%s message_id=%s consumer=%s attempt=%d error=%q", event, AnalysisStream, messageID, consumer.ConsumerID(), attempt, err.Error())
}

func observeRedisError(options RunOptions, operation string) {
	if options.Metrics != nil {
		options.Metrics.ObserveRedisError(operation)
	}
}

func observeOutboxOutcome(options RunOptions, outcome outbox.PublishOutcome) {
	if options.Metrics == nil {
		return
	}
	options.Metrics.ObserveOutboxPendingAge(outcome.PendingAge)
	switch outcome.Status {
	case outbox.PublishStatusDelivered:
		options.Metrics.ObserveOutboxPublished()
	case outbox.PublishStatusFailed:
		options.Metrics.ObserveOutboxPublishError()
	}
}

func RuntimeErrorBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<uint(attempt-1)) * 250 * time.Millisecond
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
