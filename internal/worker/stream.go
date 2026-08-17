package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

const (
	AnalysisStream     = "agentscope:analysis:v1"
	AnalysisGroup      = "agentscope-analysis"
	AnalysisDeadStream = "agentscope:analysis:dead:v1"
	AnalysisConsumer   = "agentscope-worker"
)

type StreamPublisher struct {
	client *redisv9.Client
}

func NewStreamPublisher(client *redisv9.Client) *StreamPublisher {
	return &StreamPublisher{client: client}
}

func (p *StreamPublisher) Publish(ctx context.Context, message AnalysisMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = p.client.XAdd(ctx, &redisv9.XAddArgs{
		Stream: AnalysisStream,
		Values: map[string]any{
			"version": "v1",
			"payload": string(payload),
		},
	}).Result()
	return err
}

func EnsureAnalysisGroup(ctx context.Context, client *redisv9.Client) error {
	err := client.XGroupCreateMkStream(ctx, AnalysisStream, AnalysisGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

type StreamConsumer struct {
	client *redisv9.Client
}

func NewStreamConsumer(client *redisv9.Client) *StreamConsumer {
	return &StreamConsumer{client: client}
}

func (c *StreamConsumer) Read(ctx context.Context, count int, block time.Duration) ([]redisv9.XStream, error) {
	return c.client.XReadGroup(ctx, &redisv9.XReadGroupArgs{
		Group:    AnalysisGroup,
		Consumer: AnalysisConsumer,
		Streams:  []string{AnalysisStream, ">"},
		Count:    int64(count),
		Block:    block,
	}).Result()
}

func (c *StreamConsumer) Ack(ctx context.Context, ids ...string) error {
	return c.client.XAck(ctx, AnalysisStream, AnalysisGroup, ids...).Err()
}

func (c *StreamConsumer) Requeue(ctx context.Context, message AnalysisMessage, attempt int) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.client.XAdd(ctx, &redisv9.XAddArgs{
		Stream: AnalysisStream,
		Values: map[string]any{
			"version": "v1",
			"attempt": strconv.Itoa(attempt),
			"payload": string(payload),
		},
	}).Result()
	return err
}

func (c *StreamConsumer) DeadLetter(ctx context.Context, message AnalysisMessage, attempt int) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.client.XAdd(ctx, &redisv9.XAddArgs{
		Stream: AnalysisDeadStream,
		Values: map[string]any{
			"version": "v1",
			"attempt": strconv.Itoa(attempt),
			"payload": string(payload),
		},
	}).Result()
	return err
}

func DecodeAnalysisMessage(values map[string]any) (AnalysisMessage, error) {
	raw, ok := values["payload"].(string)
	if !ok || raw == "" {
		return AnalysisMessage{}, fmt.Errorf("analysis message payload is missing")
	}
	var message AnalysisMessage
	if err := json.Unmarshal([]byte(raw), &message); err != nil {
		return AnalysisMessage{}, fmt.Errorf("decode analysis message: %w", err)
	}
	if message.Version == "" {
		message.Version = "v1"
	}
	return message, nil
}

func RetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

func AttemptFromValues(values map[string]any) int {
	raw, ok := values["attempt"].(string)
	if !ok {
		return 1
	}
	attempt, err := strconv.Atoi(raw)
	if err != nil || attempt < 1 {
		return 1
	}
	return attempt
}
