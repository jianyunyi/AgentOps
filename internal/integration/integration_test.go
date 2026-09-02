package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"agentscope/internal/outbox"
	"agentscope/internal/platform/database"
	platformredis "agentscope/internal/platform/redis"
	"agentscope/internal/trace"
	"agentscope/internal/worker"
	redisv9 "github.com/redis/go-redis/v9"
)

func TestMySQLRedisTraceOutboxFlow(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run MySQL/Redis integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.Open(ctx, os.Getenv("MYSQL_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	client := platformredis.NewClient(os.Getenv("REDIS_ADDR"))
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().UnixNano()
	tenantID, eventID := fmt.Sprintf("it_ten_%d", stamp), fmt.Sprintf("it_evt_%d", stamp)
	traceID, spanID := fmt.Sprintf("it_trc_%d", stamp), fmt.Sprintf("it_spn_%d", stamp)
	repo := trace.NewGORMRepository(db)
	_, err = repo.IngestEventAtomic(ctx, trace.IngestContext{TenantID: tenantID, AgentID: "it_agent"}, trace.Event{EventID: eventID, TraceID: traceID, SpanID: spanID, EventType: trace.EventLLMCall, OccurredAt: time.Now().UTC(), Payload: json.RawMessage(`{"input":"hello"}`)})
	if err != nil {
		t.Fatal(err)
	}

	outboxRepo := outbox.NewGORMRepository(db)
	publisher := outbox.NewPublisher(outboxRepo, func(ctx context.Context, event outbox.Event) error {
		var message worker.AnalysisMessage
		if err := json.Unmarshal(event.Payload, &message); err != nil {
			return err
		}
		return worker.NewStreamPublisher(client).Publish(ctx, message)
	})
	if err := publisher.PublishOne(ctx); err != nil {
		t.Fatal(err)
	}
	result, err := client.XRange(ctx, worker.AnalysisStream, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected analysis message in Redis Stream")
	}
}

func TestRedisStreamPendingRecovery(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run MySQL/Redis integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := platformredis.NewClient(os.Getenv("REDIS_ADDR"))
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().UnixNano()
	stream := fmt.Sprintf("agentscope:it:worker:%d", stamp)
	group := fmt.Sprintf("agentscope-it-group-%d", stamp)
	firstConsumer := fmt.Sprintf("first-%d", stamp)
	recoveryConsumer := fmt.Sprintf("recovery-%d", stamp)
	defer client.Del(context.Background(), stream)
	if _, err := client.XAdd(ctx, &redisv9.XAddArgs{Stream: stream, Values: map[string]any{"payload": "safe"}}).Result(); err != nil {
		t.Fatal(err)
	}
	if err := client.XGroupCreate(ctx, stream, group, "0").Err(); err != nil {
		t.Fatal(err)
	}
	claimed, err := client.XReadGroup(ctx, &redisv9.XReadGroupArgs{Group: group, Consumer: firstConsumer, Streams: []string{stream, ">"}, Count: 1, Block: time.Second}).Result()
	if err != nil || len(claimed) != 1 || len(claimed[0].Messages) != 1 {
		t.Fatalf("initial group read = batches:%d err:%v", len(claimed), err)
	}

	messages, _, err := client.XAutoClaim(ctx, &redisv9.XAutoClaimArgs{Stream: stream, Group: group, Consumer: recoveryConsumer, MinIdle: time.Millisecond, Start: "0-0", Count: 10}).Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].ID != claimed[0].Messages[0].ID {
		t.Fatalf("recovered messages = %+v, initial = %+v", messages, claimed)
	}
	if _, err := client.XAck(ctx, stream, group, messages[0].ID).Result(); err != nil {
		t.Fatal(err)
	}
}
