package trace

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
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
