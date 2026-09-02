package worker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	client      streamClient
	consumer    string
	pendingIdle time.Duration
}

func NewStreamConsumer(client *redisv9.Client) *StreamConsumer {
	return NewConfiguredStreamConsumer(client, "", 2*time.Minute)
}

func NewConfiguredStreamConsumer(client *redisv9.Client, consumerID string, pendingIdle time.Duration) *StreamConsumer {
	if pendingIdle <= 0 {
		pendingIdle = 2 * time.Minute
	}
	return &StreamConsumer{client: redisStreamClient{client: client}, consumer: NewConsumerID(consumerID), pendingIdle: pendingIdle}
}

func NewConsumerID(override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	hostname = strings.NewReplacer(" ", "-", "/", "-", "\\", "-").Replace(hostname)
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("agentscope-worker-%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("agentscope-worker-%s-%d-%x", hostname, os.Getpid(), random)
}

func (c *StreamConsumer) EnsureGroup(ctx context.Context) error {
	return c.client.createGroup(ctx, AnalysisStream, AnalysisGroup, "0")
}

func (c *StreamConsumer) Read(ctx context.Context, count int, block time.Duration) ([]redisv9.XStream, error) {
	return c.client.readGroup(ctx, AnalysisGroup, c.consumer, count, block)
}

func (c *StreamConsumer) Ack(ctx context.Context, ids ...string) error {
	_, err := c.client.ack(ctx, AnalysisStream, AnalysisGroup, ids...)
	return err
}

func (c *StreamConsumer) ClaimPending(ctx context.Context, count int) ([]redisv9.XMessage, error) {
	messages, _, err := c.client.autoClaim(ctx, AnalysisGroup, c.consumer, c.pendingIdle, "0-0", count)
	return messages, err
}

func (c *StreamConsumer) Requeue(ctx context.Context, message AnalysisMessage, attempt int) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.client.add(ctx, AnalysisStream, map[string]any{
		"version": "v1",
		"attempt": strconv.Itoa(attempt),
		"payload": string(payload),
	})
	return err
}

func (c *StreamConsumer) DeadLetter(ctx context.Context, message AnalysisMessage, attempt int) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = c.client.add(ctx, AnalysisDeadStream, map[string]any{
		"version": "v1",
		"attempt": strconv.Itoa(attempt),
		"payload": string(payload),
	})
	return err
}

type streamClient interface {
	createGroup(context.Context, string, string, string) error
	readGroup(context.Context, string, string, int, time.Duration) ([]redisv9.XStream, error)
	autoClaim(context.Context, string, string, time.Duration, string, int) ([]redisv9.XMessage, string, error)
	ack(context.Context, string, string, ...string) (int64, error)
	add(context.Context, string, map[string]any) (string, error)
}

type redisStreamClient struct{ client *redisv9.Client }

func (c redisStreamClient) createGroup(ctx context.Context, stream, group, start string) error {
	err := c.client.XGroupCreateMkStream(ctx, stream, group, start).Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c redisStreamClient) readGroup(ctx context.Context, group, consumer string, count int, block time.Duration) ([]redisv9.XStream, error) {
	return c.client.XReadGroup(ctx, &redisv9.XReadGroupArgs{Group: group, Consumer: consumer, Streams: []string{AnalysisStream, ">"}, Count: int64(count), Block: block}).Result()
}

func (c redisStreamClient) autoClaim(ctx context.Context, group, consumer string, minIdle time.Duration, start string, count int) ([]redisv9.XMessage, string, error) {
	return c.client.XAutoClaim(ctx, &redisv9.XAutoClaimArgs{Stream: AnalysisStream, Group: group, Consumer: consumer, MinIdle: minIdle, Start: start, Count: int64(count)}).Result()
}

func (c redisStreamClient) ack(ctx context.Context, stream, group string, ids ...string) (int64, error) {
	return c.client.XAck(ctx, stream, group, ids...).Result()
}

func (c redisStreamClient) add(ctx context.Context, stream string, values map[string]any) (string, error) {
	return c.client.XAdd(ctx, &redisv9.XAddArgs{Stream: stream, Values: values}).Result()
}

func (c *StreamConsumer) ConsumerID() string { return c.consumer }

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
