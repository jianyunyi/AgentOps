package risk

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, event *RiskEvent) error
	List(ctx context.Context, tenantID string, offset, limit int, status string) ([]RiskEvent, int64, error)
	UpdateStatus(ctx context.Context, tenantID, id, status string) error
}

type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) Create(ctx context.Context, event *RiskEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *GORMRepository) List(ctx context.Context, tenantID string, offset, limit int, status string) ([]RiskEvent, int64, error) {
	var events []RiskEvent
	var total int64
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Model(&RiskEvent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *GORMRepository) UpdateStatus(ctx context.Context, tenantID, id, status string) error {
	return r.db.WithContext(ctx).Model(&RiskEvent{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("status", status).Error
}

func IsDuplicate(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
