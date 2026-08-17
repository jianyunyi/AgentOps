package trace

import (
	"context"
	"errors"

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
