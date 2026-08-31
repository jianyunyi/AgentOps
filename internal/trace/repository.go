package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"agentscope/internal/outbox"
	"gorm.io/gorm"
)

var ErrTraceNotFound = errors.New("trace not found")

type Repository interface {
	EventExists(ctx context.Context, tenantID, eventID string) (bool, error)
	FindTrace(ctx context.Context, tenantID, traceID string) (*Trace, error)
	CreateTrace(ctx context.Context, trace *Trace) error
	UpdateTrace(ctx context.Context, trace *Trace) error
	CreateSpan(ctx context.Context, span *Span) error
	MarkEvent(ctx context.Context, tenantID, eventID string) error
}

type QueryRepository interface {
	FindTrace(ctx context.Context, tenantID, traceID string) (*Trace, error)
	ListTraces(ctx context.Context, tenantID string, offset, limit int) ([]Trace, int64, error)
	ListSpans(ctx context.Context, tenantID, traceID string) ([]Span, error)
}

type GORMRepository struct {
	db *gorm.DB
}

func NewGORMRepository(db *gorm.DB) *GORMRepository {
	return &GORMRepository{db: db}
}

func (r *GORMRepository) EventExists(ctx context.Context, tenantID, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("trace_events").Where("tenant_id = ? AND event_id = ?", tenantID, eventID).Count(&count).Error
	return count > 0, err
}

func (r *GORMRepository) FindTrace(ctx context.Context, tenantID, traceID string) (*Trace, error) {
	var trace Trace
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND trace_id = ?", tenantID, traceID).First(&trace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTraceNotFound
		}
		return nil, err
	}
	return &trace, nil
}

func (r *GORMRepository) CreateTrace(ctx context.Context, trace *Trace) error {
	return r.db.WithContext(ctx).Create(trace).Error
}

func (r *GORMRepository) UpdateTrace(ctx context.Context, trace *Trace) error {
	return r.db.WithContext(ctx).Save(trace).Error
}

func (r *GORMRepository) CreateSpan(ctx context.Context, span *Span) error {
	return r.db.WithContext(ctx).Create(span).Error
}

func (r *GORMRepository) MarkEvent(ctx context.Context, tenantID, eventID string) error {
	return r.db.WithContext(ctx).Table("trace_events").Create(map[string]any{
		"tenant_id": tenantID,
		"event_id":  eventID,
	}).Error
}

func (r *GORMRepository) IngestEventAtomic(ctx context.Context, identity IngestContext, event Event) (IngestResult, error) {
	var duplicate bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&EventRecord{}).Where("tenant_id = ? AND event_id = ?", identity.TenantID, event.EventID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			duplicate = true
			return nil
		}
		now := time.Now().UTC()
		var trace Trace
		traceErr := tx.Where("tenant_id = ? AND trace_id = ?", identity.TenantID, event.TraceID).First(&trace).Error
		if errors.Is(traceErr, gorm.ErrRecordNotFound) {
			trace = Trace{ID: traceID(), TenantID: identity.TenantID, AgentID: identity.AgentID, TraceID: event.TraceID, Status: TraceRunning, RiskLevel: "none", StartedAt: event.OccurredAt, AnalysisStatus: "pending", CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&trace).Error; err != nil {
				return err
			}
		} else if traceErr != nil {
			return traceErr
		}
		span := Span{ID: spanID(), TenantID: identity.TenantID, TraceID: event.TraceID, SpanID: event.SpanID, ParentSpanID: event.ParentSpanID, SpanType: event.EventType, Name: event.EventType, Status: "success", Sequence: event.Sequence, InputSnapshot: event.Payload, StartedAt: event.OccurredAt, CreatedAt: now}
		if err := tx.Create(&span).Error; err != nil {
			return err
		}
		if event.EventType == EventTraceEnd {
			trace.Status = TraceSuccess
			trace.EndedAt = &event.OccurredAt
			trace.DurationMS = event.OccurredAt.Sub(trace.StartedAt).Milliseconds()
			trace.UpdatedAt = now
			if err := tx.Save(&trace).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&EventRecord{TenantID: identity.TenantID, EventID: event.EventID, CreatedAt: now}).Error; err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]string{"tenant_id": identity.TenantID, "event_id": event.EventID, "trace_id": event.TraceID, "span_id": event.SpanID})
		if err != nil {
			return err
		}
		outboxRecord := outbox.Event{ID: outboxID(), TenantID: identity.TenantID, EventType: "trace.analyze", AggregateID: event.TraceID, Payload: payload, Status: outbox.StatusPending, AvailableAt: now, CreatedAt: now}
		return tx.Create(&outboxRecord).Error
	})
	if err != nil {
		return IngestResult{}, err
	}
	return IngestResult{Duplicate: duplicate}, nil
}

func traceID() string  { return randomPrefixedID("trc_") }
func spanID() string   { return randomPrefixedID("spn_") }
func outboxID() string { return randomPrefixedID("out_") }
func randomPrefixedID(prefix string) string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(buf)
}

func (r *GORMRepository) ListTraces(ctx context.Context, tenantID string, offset, limit int) ([]Trace, int64, error) {
	var traces []Trace
	var total int64
	query := r.db.WithContext(ctx).Model(&Trace{}).Where("tenant_id = ?", tenantID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&traces).Error; err != nil {
		return nil, 0, err
	}
	return traces, total, nil
}

func (r *GORMRepository) ListSpans(ctx context.Context, tenantID, traceID string) ([]Span, error) {
	var spans []Span
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND trace_id = ?", tenantID, traceID).Order("sequence ASC").Find(&spans).Error; err != nil {
		return nil, err
	}
	return spans, nil
}
