package worker

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

type fakeStreamClient struct {
	groupErr       error
	claimResults   []fakeClaimResult
	readResults    []fakeReadResult
	acks           []string
	added          []fakeAddCall
	callOrder      []string
	ackErr         error
	addErrByStream map[string]error
	readConsumer   string
	claimConsumer  string
}

type fakeClaimResult struct {
	messages []redisv9.XMessage
	next     string
	err      error
}

type fakeReadResult struct {
	streams []redisv9.XStream
	err     error
}

type fakeAddCall struct {
	stream string
	values map[string]any
}

func (f *fakeStreamClient) createGroup(context.Context, string, string, string) error {
	f.callOrder = append(f.callOrder, "group")
	return f.groupErr
}

func (f *fakeStreamClient) readGroup(_ context.Context, _ string, consumer string, _ int, _ time.Duration) ([]redisv9.XStream, error) {
	f.callOrder = append(f.callOrder, "read")
	f.readConsumer = consumer
	if len(f.readResults) == 0 {
		return nil, context.Canceled
	}
	result := f.readResults[0]
	f.readResults = f.readResults[1:]
	return result.streams, result.err
}

func (f *fakeStreamClient) autoClaim(_ context.Context, _ string, consumer string, _ time.Duration, _ string, _ int) ([]redisv9.XMessage, string, error) {
	f.callOrder = append(f.callOrder, "claim")
	f.claimConsumer = consumer
	if len(f.claimResults) == 0 {
		return nil, "0-0", nil
	}
	result := f.claimResults[0]
	f.claimResults = f.claimResults[1:]
	return result.messages, result.next, result.err
}

func (f *fakeStreamClient) ack(_ context.Context, _ string, _ string, ids ...string) (int64, error) {
	f.callOrder = append(f.callOrder, "ack:"+ids[0])
	if f.ackErr != nil {
		return 0, f.ackErr
	}
	f.acks = append(f.acks, ids...)
	return int64(len(ids)), nil
}

func (f *fakeStreamClient) add(_ context.Context, stream string, values map[string]any) (string, error) {
	f.callOrder = append(f.callOrder, "add:"+stream)
	f.added = append(f.added, fakeAddCall{stream: stream, values: values})
	if err := f.addErrByStream[stream]; err != nil {
		return "", err
	}
	return "new-id", nil
}

func testConsumer(fake *fakeStreamClient) *StreamConsumer {
	return &StreamConsumer{client: fake, consumer: "worker-test", pendingIdle: time.Minute}
}

func noWaitOptions() RunOptions {
	return RunOptions{
		Block:        time.Nanosecond,
		ErrorBackoff: func(int) time.Duration { return 0 },
		Sleep:        func(context.Context, time.Duration) error { return nil },
		Logger:       func(string, ...any) {},
	}
}

func message(id string) redisv9.XMessage {
	return messageWith(id, id)
}

func messageWith(id, eventID string) redisv9.XMessage {
	return redisv9.XMessage{ID: id, Values: map[string]any{
		"version": "v1",
		"attempt": "1",
		"payload": `{"event_id":"` + eventID + `","tenant_id":"ten-1"}`,
	}}
}

func streamWith(message redisv9.XMessage) []redisv9.XStream {
	return []redisv9.XStream{{Stream: AnalysisStream, Messages: []redisv9.XMessage{message}}}
}

func TestNewConsumerIDUsesOverrideOrUniqueGeneratedValue(t *testing.T) {
	if got := NewConsumerID("explicit-consumer"); got != "explicit-consumer" {
		t.Fatalf("override consumer = %q", got)
	}
	first, second := NewConsumerID(""), NewConsumerID("")
	if first == second || first == "" || second == "" {
		t.Fatalf("generated consumers must be non-empty and unique: %q, %q", first, second)
	}
}

func TestDecodeAnalysisMessageReadsOutboxPayload(t *testing.T) {
	got, err := DecodeAnalysisMessage(map[string]any{
		"payload": `{"tenant_id":"ten-1","event_id":"evt-1","trace_id":"trace-1","span_id":"span-1","input":"safe"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TenantID != "ten-1" || got.EventID != "evt-1" || got.TraceID != "trace-1" || got.SpanID != "span-1" || got.Input != "safe" {
		t.Fatalf("decoded analysis message = %+v", got)
	}
}

func TestRunUsesConfiguredConsumerForReadAndRecovery(t *testing.T) {
	fake := &fakeStreamClient{
		claimResults: []fakeClaimResult{{messages: nil}},
		readResults:  []fakeReadResult{{err: context.Canceled}},
	}
	consumer := testConsumer(fake)
	consumer.consumer = "configured-consumer"
	err := RunWithOptions(context.Background(), consumer, NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error { return nil }), noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if fake.claimConsumer != "configured-consumer" || fake.readConsumer != "configured-consumer" {
		t.Fatalf("consumer IDs = claim:%q read:%q", fake.claimConsumer, fake.readConsumer)
	}
}

func TestRunProcessesPendingBeforeNewMessages(t *testing.T) {
	fake := &fakeStreamClient{
		claimResults: []fakeClaimResult{{messages: []redisv9.XMessage{messageWith("pending-1", "evt-pending")}}},
		readResults:  []fakeReadResult{{streams: streamWith(messageWith("new-1", "evt-new")), err: nil}, {err: context.Canceled}},
	}
	consumer := testConsumer(fake)
	processed := make([]string, 0, 2)
	processor := NewAnalysisProcessor(3, func(_ context.Context, msg AnalysisMessage) error {
		processed = append(processed, msg.EventID)
		return nil
	})

	err := RunWithOptions(context.Background(), consumer, processor, noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if !reflect.DeepEqual(fake.acks, []string{"pending-1", "new-1"}) {
		t.Fatalf("acked ids = %v", fake.acks)
	}
	if !reflect.DeepEqual(processed, []string{"evt-pending", "evt-new"}) {
		t.Fatalf("processed event ids = %v", processed)
	}
	if !reflect.DeepEqual(fake.callOrder[:4], []string{"group", "claim", "ack:pending-1", "read"}) {
		t.Fatalf("call order = %v", fake.callOrder)
	}
}

func TestRunRetriesWithoutAckWhenRequeueFails(t *testing.T) {
	fake := &fakeStreamClient{
		readResults:    []fakeReadResult{{streams: streamWith(message("new-1")), err: nil}, {err: context.Canceled}},
		addErrByStream: map[string]error{AnalysisStream: errors.New("redis unavailable")},
	}
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return errors.New("temporary analyzer failure")
	})

	err := RunWithOptions(context.Background(), testConsumer(fake), processor, noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if len(fake.acks) != 0 {
		t.Fatalf("message was ACKed after failed requeue: %v", fake.acks)
	}
	if len(fake.added) != 1 || fake.added[0].stream != AnalysisStream {
		t.Fatalf("requeue calls = %+v", fake.added)
	}
}

func TestRunRequeuesThenAcksAfterRetryTargetSucceeds(t *testing.T) {
	fake := &fakeStreamClient{
		readResults:    []fakeReadResult{{streams: streamWith(message("new-1")), err: nil}, {err: context.Canceled}},
		addErrByStream: map[string]error{},
	}
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return errors.New("temporary analyzer failure")
	})
	err := RunWithOptions(context.Background(), testConsumer(fake), processor, noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if !reflect.DeepEqual(fake.acks, []string{"new-1"}) {
		t.Fatalf("acked ids = %v", fake.acks)
	}
	if len(fake.added) != 1 || fake.added[0].stream != AnalysisStream || fake.added[0].values["attempt"] != "2" {
		t.Fatalf("retry message = %+v", fake.added)
	}
}

func TestRunDeadLettersThenAcksAfterMaximumAttempts(t *testing.T) {
	item := message("new-1")
	item.Values["attempt"] = "3"
	fake := &fakeStreamClient{
		readResults:    []fakeReadResult{{streams: streamWith(item), err: nil}, {err: context.Canceled}},
		addErrByStream: map[string]error{},
	}
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return errors.New("permanent analyzer failure")
	})
	err := RunWithOptions(context.Background(), testConsumer(fake), processor, noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if !reflect.DeepEqual(fake.acks, []string{"new-1"}) {
		t.Fatalf("acked ids = %v", fake.acks)
	}
	if len(fake.added) != 1 || fake.added[0].stream != AnalysisDeadStream {
		t.Fatalf("dead letter message = %+v", fake.added)
	}
}

func TestRunDeadLettersBeforeAckAndLeavesPendingOnDLQFailure(t *testing.T) {
	fake := &fakeStreamClient{
		readResults:    []fakeReadResult{{streams: streamWith(redisv9.XMessage{ID: "bad-1", Values: map[string]any{"attempt": "3"}}), err: nil}, {err: context.Canceled}},
		addErrByStream: map[string]error{AnalysisDeadStream: errors.New("redis unavailable")},
	}
	err := RunWithOptions(context.Background(), testConsumer(fake), NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error { return nil }), noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if len(fake.acks) != 0 {
		t.Fatalf("malformed message was ACKed after failed DLQ: %v", fake.acks)
	}
}

func TestRunContinuesAfterRuntimeRedisError(t *testing.T) {
	fake := &fakeStreamClient{
		claimResults: []fakeClaimResult{{err: errors.New("temporary redis error")}, {messages: nil}},
		readResults:  []fakeReadResult{{err: context.Canceled}},
	}
	err := RunWithOptions(context.Background(), testConsumer(fake), NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error { return nil }), noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if len(fake.callOrder) < 4 || fake.callOrder[0] != "group" || fake.callOrder[1] != "claim" || fake.callOrder[2] != "claim" || fake.callOrder[3] != "read" {
		t.Fatalf("runtime error stopped loop: call order = %v", fake.callOrder)
	}
}

func TestRunOutboxContinuesAfterRuntimeError(t *testing.T) {
	publisher := &fakeOutboxPublisher{results: []error{errors.New("database unavailable"), nil}}
	ctx, cancel := context.WithCancel(context.Background())
	publisher.onSuccess = cancel
	err := RunOutboxWithOptions(ctx, publisher.publish, time.Nanosecond, noWaitOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunOutbox() error = %v, want context canceled", err)
	}
	if publisher.calls != 2 {
		t.Fatalf("PublishOne calls = %d, want 2", publisher.calls)
	}
}

type fakeOutboxPublisher struct {
	results   []error
	calls     int
	onSuccess func()
}

func (f *fakeOutboxPublisher) publish(ctx context.Context) error {
	f.calls++
	if len(f.results) == 0 {
		return context.Canceled
	}
	err := f.results[0]
	f.results = f.results[1:]
	if err == nil && f.onSuccess != nil {
		f.onSuccess()
	}
	return err
}
