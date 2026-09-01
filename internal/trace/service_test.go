package trace

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agentscope/internal/outbox"
)

type fakeTraceRepository struct {
	eventIDs map[string]bool
	traces   []*Trace
	spans    []*Span
}

func (f *fakeTraceRepository) FindTrace(_ context.Context, tenantID, traceID string) (*Trace, error) {
	for _, trace := range f.traces {
		if trace.TenantID == tenantID && trace.TraceID == traceID {
			return trace, nil
		}
	}
	return nil, ErrTraceNotFound
}

func (f *fakeTraceRepository) EventExists(_ context.Context, tenantID, eventID string) (bool, error) {
	return f.eventIDs[tenantID+":"+eventID], nil
}

func (f *fakeTraceRepository) CreateTrace(_ context.Context, trace *Trace) error {
	f.traces = append(f.traces, trace)
	return nil
}

func (f *fakeTraceRepository) CreateSpan(_ context.Context, span *Span) error {
	f.spans = append(f.spans, span)
	return nil
}

func (f *fakeTraceRepository) UpdateTrace(_ context.Context, trace *Trace) error {
	for i, current := range f.traces {
		if current.TenantID == trace.TenantID && current.TraceID == trace.TraceID {
			f.traces[i] = trace
			return nil
		}
	}
	return ErrTraceNotFound
}

func (f *fakeTraceRepository) MarkEvent(_ context.Context, tenantID, eventID string) error {
	if f.eventIDs == nil {
		f.eventIDs = map[string]bool{}
	}
	f.eventIDs[tenantID+":"+eventID] = true
	return nil
}

type fakeTraceOutboxRepository struct{ events []outbox.Event }

func (f *fakeTraceOutboxRepository) Create(_ context.Context, event *outbox.Event) error {
	f.events = append(f.events, *event)
	return nil
}

func (f *fakeTraceOutboxRepository) ClaimPending(context.Context) (*outbox.Event, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTraceOutboxRepository) MarkDelivered(context.Context, string) error { return nil }
func (f *fakeTraceOutboxRepository) MarkFailed(context.Context, string) error    { return nil }

func TestIngestPersistsOnlyRedactedPayload(t *testing.T) {
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	outboxRepo := &fakeTraceOutboxRepository{}
	svc := NewServiceWithOutbox(repo, outbox.NewService(outboxRepo))
	event := Event{
		EventID:    "evt_redact",
		TraceID:    "trace_redact",
		SpanID:     "span_redact",
		EventType:  EventLLMCall,
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"prompt":"contact user@example.com","api_key":"sk-live-1234567890"}`),
	}

	if _, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, event); err != nil {
		t.Fatal(err)
	}
	if len(repo.spans) != 1 || strings.Contains(string(repo.spans[0].InputSnapshot), "user@example.com") || strings.Contains(string(repo.spans[0].InputSnapshot), "sk-live-1234567890") {
		t.Fatalf("span payload leaked original data: %+v", repo.spans)
	}
	if len(outboxRepo.events) != 1 || strings.Contains(string(outboxRepo.events[0].Payload), "user@example.com") || strings.Contains(string(outboxRepo.events[0].Payload), "sk-live-1234567890") {
		t.Fatalf("outbox payload leaked original data: %+v", outboxRepo.events)
	}
}

func TestIngestRejectsMalformedPayloadBeforePersistence(t *testing.T) {
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	svc := NewService(repo)
	_, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, Event{
		EventID:    "evt_invalid_payload",
		TraceID:    "trace_invalid_payload",
		SpanID:     "span_invalid_payload",
		EventType:  EventLLMCall,
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"prompt":`),
	})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("error = %v, want ErrInvalidPayload", err)
	}
	if len(repo.traces) != 0 || len(repo.spans) != 0 {
		t.Fatal("malformed payload must not persist trace or span")
	}
}

func TestIngestAcceptsTraceStartAndCreatesRunningTrace(t *testing.T) {
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	svc := NewService(repo)
	event := Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  EventTraceStart,
		Sequence:   1,
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"input":"check order errors"}`),
	}

	result, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, event)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Duplicate {
		t.Fatal("first event must not be marked duplicate")
	}
	if len(repo.traces) != 1 || repo.traces[0].Status != TraceRunning {
		t.Fatalf("expected one running trace, got %+v", repo.traces)
	}
	if len(repo.spans) != 1 || repo.spans[0].TenantID != "tenant_001" {
		t.Fatalf("expected one tenant-bound span, got %+v", repo.spans)
	}
}

func TestIngestRejectsUnknownEventType(t *testing.T) {
	svc := NewService(&fakeTraceRepository{eventIDs: map[string]bool{}})
	_, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  "unknown",
		OccurredAt: time.Now().UTC(),
	})
	if !errors.Is(err, ErrInvalidEventType) {
		t.Fatalf("expected ErrInvalidEventType, got %v", err)
	}
}

func TestIngestRejectsFutureEvent(t *testing.T) {
	svc := NewService(&fakeTraceRepository{eventIDs: map[string]bool{}})
	_, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  EventTraceStart,
		OccurredAt: time.Now().UTC().Add(2 * time.Minute),
	})
	if !errors.Is(err, ErrEventTimeInvalid) {
		t.Fatalf("expected ErrEventTimeInvalid, got %v", err)
	}
}

func TestIngestRejectsOversizedPayload(t *testing.T) {
	svc := NewService(&fakeTraceRepository{eventIDs: map[string]bool{}})
	_, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  EventTraceStart,
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(make([]byte, MaxPayloadBytes+1)),
	})
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestIngestReturnsIdempotentResultForDuplicateEvent(t *testing.T) {
	repo := &fakeTraceRepository{eventIDs: map[string]bool{"tenant_001:evt_001": true}}
	svc := NewService(repo)
	result, err := svc.Ingest(context.Background(), IngestContext{TenantID: "tenant_001", AgentID: "agent_001"}, Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  EventTraceStart,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("duplicate Ingest() error = %v", err)
	}
	if !result.Duplicate {
		t.Fatal("duplicate event must be accepted idempotently")
	}
	if len(repo.traces) != 0 || len(repo.spans) != 0 {
		t.Fatal("duplicate event must not create new rows")
	}
}
