package audit

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Append(context.Context, *Record) error
}
type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }
func (r *GORMRepository) Append(ctx context.Context, record *Record) error {
	return r.db.WithContext(ctx).Create(record).Error
}
