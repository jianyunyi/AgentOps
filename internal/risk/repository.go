package risk

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, event *RiskEvent) error
}

type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) Create(ctx context.Context, event *RiskEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func IsDuplicate(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
