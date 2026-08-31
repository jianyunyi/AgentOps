package outbox

import (
	"context"
	"gorm.io/gorm"
	"time"
)

type Repository interface {
	Create(context.Context, *Event) error
	ClaimPending(context.Context) (*Event, error)
	MarkDelivered(context.Context, string) error
	MarkFailed(context.Context, string) error
}
type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }
func (r *GORMRepository) Create(ctx context.Context, event *Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *GORMRepository) ClaimPending(ctx context.Context) (*Event, error) {
	var event Event
	err := r.db.WithContext(ctx).Where("status = ? AND available_at <= ?", StatusPending, time.Now().UTC()).Order("created_at ASC").First(&event).Error
	return &event, err
}

func (r *GORMRepository) MarkDelivered(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Event{}).Where("id = ?", id).Updates(map[string]any{"status": StatusDelivered, "delivered_at": now}).Error
}

func (r *GORMRepository) MarkFailed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&Event{}).Where("id = ?", id).Updates(map[string]any{"attempts": gorm.Expr("attempts + 1"), "available_at": time.Now().UTC().Add(5 * time.Second)}).Error
}
