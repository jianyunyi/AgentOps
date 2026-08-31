package outbox

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Create(context.Context, *Event) error
}
type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }
func (r *GORMRepository) Create(ctx context.Context, event *Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}
