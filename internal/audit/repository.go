package audit

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Append(context.Context, *Record) error
	List(context.Context, string, int, int, string) ([]Record, int64, error)
}

func (r *GORMRepository) List(ctx context.Context, tenantID string, offset, limit int, action string) ([]Record, int64, error) {
	var records []Record
	var total int64
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Model(&Record{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }
func (r *GORMRepository) Append(ctx context.Context, record *Record) error {
	return r.db.WithContext(ctx).Create(record).Error
}
