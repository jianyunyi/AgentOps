package worker

import (
	"context"
	"time"

	"agentscope/internal/outbox"
	redisv9 "github.com/redis/go-redis/v9"
)

func RunOutbox(ctx context.Context, publisher *outbox.Publisher, interval time.Duration) error {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := publisher.PublishOne(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func Run(ctx context.Context, consumer *StreamConsumer, processor *AnalysisProcessor) error {
	if err := EnsureAnalysisGroup(ctx, consumer.client); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		batches, err := consumer.Read(ctx, 10, 5*time.Second)
		if err == redisv9.Nil {
			continue
		}
		if err != nil {
			return err
		}
		for _, batch := range batches {
			for _, item := range batch.Messages {
				message, err := DecodeAnalysisMessage(item.Values)
				if err != nil {
					if deadErr := consumer.DeadLetter(ctx, AnalysisMessage{EventID: item.ID}, AttemptFromValues(item.Values)); deadErr != nil {
						return deadErr
					}
					if err := consumer.Ack(ctx, item.ID); err != nil {
						return err
					}
					continue
				}
				attempt := AttemptFromValues(item.Values)
				decision := processor.Process(ctx, message, attempt)
				switch decision {
				case DecisionAck:
					if err := consumer.Ack(ctx, item.ID); err != nil {
						return err
					}
				case DecisionRetry:
					if err := consumer.Requeue(ctx, message, attempt+1); err != nil {
						return err
					}
					if err := consumer.Ack(ctx, item.ID); err != nil {
						return err
					}
				case DecisionDeadLetter:
					if err := consumer.DeadLetter(ctx, message, attempt); err != nil {
						return err
					}
					if err := consumer.Ack(ctx, item.ID); err != nil {
						return err
					}
				}
			}
		}
	}
}
