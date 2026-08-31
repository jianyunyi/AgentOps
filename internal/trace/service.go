package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"agentscope/internal/outbox"
)

var (
	ErrInvalidEventType = errors.New("invalid event type")
	ErrEventTimeInvalid = errors.New("event timestamp is outside accepted window")
	ErrPayloadTooLarge  = errors.New("event payload is too large")
)

type Service struct {
	repo   Repository
	outbox *outbox.Service
}

type atomicIngestRepository interface {
	IngestEventAtomic(ctx context.Context, identity IngestContext, event Event) (IngestResult, error)
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithOutbox(repo Repository, outboxService *outbox.Service) *Service {
	return &Service{repo: repo, outbox: outboxService}
}

func (s *Service) Ingest(ctx context.Context, identity IngestContext, event Event) (IngestResult, error) {
	if strings.TrimSpace(identity.TenantID) == "" || strings.TrimSpace(identity.AgentID) == "" || event.EventID == "" || event.TraceID == "" || event.SpanID == "" {
		return IngestResult{}, errors.New("tenant, agent, event, trace, and span ids are required")
	}
	if !validEventType(event.EventType) {
		return IngestResult{}, ErrInvalidEventType
	}
	if event.OccurredAt.IsZero() || event.OccurredAt.After(time.Now().UTC().Add(time.Minute)) {
		return IngestResult{}, ErrEventTimeInvalid
	}
	if len(event.Payload) > MaxPayloadBytes {
		return IngestResult{}, ErrPayloadTooLarge
	}
	if atomicRepo, ok := s.repo.(atomicIngestRepository); ok {
		return atomicRepo.IngestEventAtomic(ctx, identity, event)
	}
	exists, err := s.repo.EventExists(ctx, identity.TenantID, event.EventID)
	if err != nil {
		return IngestResult{}, err
	}
	if exists {
		return IngestResult{Duplicate: true}, nil
	}

	now := time.Now().UTC()
	trace, err := s.repo.FindTrace(ctx, identity.TenantID, event.TraceID)
	if errors.Is(err, ErrTraceNotFound) {
		trace = &Trace{
			ID:             mustID("trc_"),
			TenantID:       identity.TenantID,
			AgentID:        identity.AgentID,
			TraceID:        event.TraceID,
			Status:         TraceRunning,
			RiskLevel:      "none",
			StartedAt:      event.OccurredAt,
			AnalysisStatus: "pending",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.repo.CreateTrace(ctx, trace); err != nil {
			return IngestResult{}, err
		}
	} else if err != nil {
		return IngestResult{}, err
	}

	span := &Span{
		ID:            mustID("spn_"),
		TenantID:      identity.TenantID,
		TraceID:       event.TraceID,
		SpanID:        event.SpanID,
		ParentSpanID:  event.ParentSpanID,
		SpanType:      event.EventType,
		Name:          event.EventType,
		Status:        "success",
		Sequence:      event.Sequence,
		InputSnapshot: event.Payload,
		StartedAt:     event.OccurredAt,
		CreatedAt:     now,
	}
	if err := s.repo.CreateSpan(ctx, span); err != nil {
		return IngestResult{}, err
	}
	if event.EventType == EventTraceEnd {
		trace.Status = TraceSuccess
		trace.EndedAt = &event.OccurredAt
		trace.DurationMS = event.OccurredAt.Sub(trace.StartedAt).Milliseconds()
		trace.UpdatedAt = now
		if err := s.repo.UpdateTrace(ctx, trace); err != nil {
			return IngestResult{}, err
		}
	}
	if err := s.repo.MarkEvent(ctx, identity.TenantID, event.EventID); err != nil {
		return IngestResult{}, err
	}
	if s.outbox != nil {
		payload, _ := json.Marshal(map[string]string{"version": "v1", "tenant_id": identity.TenantID, "event_id": event.EventID, "trace_id": event.TraceID, "span_id": event.SpanID, "input": string(event.Payload)})
		if err := s.outbox.Enqueue(ctx, outbox.EventInput{TenantID: identity.TenantID, EventType: "trace.analyze", AggregateID: event.TraceID, DedupKey: identity.TenantID + ":" + event.EventID, Payload: payload}); err != nil {
			return IngestResult{}, err
		}
	}
	return IngestResult{}, nil
}

func validEventType(eventType string) bool {
	switch eventType {
	case EventTraceStart, EventLLMCall, EventToolCall, EventRiskCheck, EventAgentOutput, EventTraceEnd:
		return true
	default:
		return false
	}
}

func mustID(prefix string) string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(buf)
}
