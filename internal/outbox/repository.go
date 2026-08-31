package outbox

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxAttempts = 10
const ClaimTimeout = 5 * time.Minute

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
	workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("((status = ? AND available_at <= ?) OR (status = ? AND claimed_at < ?))", StatusPending, now, StatusProcessing, now.Add(-ClaimTimeout)).Order("created_at ASC").First(&event).Error
		if err != nil {
			return err
		}
		now = time.Now().UTC()
		return tx.Model(&Event{}).Where("id = ? AND (status = ? OR status = ?)", event.ID, StatusPending, StatusProcessing).Updates(map[string]any{"status": StatusProcessing, "claimed_at": now, "claimed_by": workerID}).Error
	})
	return &event, err
}

func (r *GORMRepository) MarkDelivered(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Event{}).Where("id = ? AND status = ?", id, StatusProcessing).Updates(map[string]any{"status": StatusDelivered, "delivered_at": now, "claimed_at": nil, "claimed_by": ""}).Error
}

func (r *GORMRepository) MarkFailed(ctx context.Context, id string) error {
	var event Event
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error; err != nil {
		return err
	}
	status := StatusPending
	if event.Attempts+1 >= MaxAttempts {
		status = StatusDead
	}
	return r.db.WithContext(ctx).Model(&Event{}).Where("id = ? AND status = ?", id, StatusProcessing).Updates(map[string]any{"status": status, "attempts": gorm.Expr("attempts + 1"), "available_at": time.Now().UTC().Add(5 * time.Second), "claimed_at": nil, "claimed_by": ""}).Error
}
