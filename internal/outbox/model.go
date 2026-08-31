package outbox

import "time"

const (
	StatusPending   = "pending"
	StatusDelivered = "delivered"
	StatusDead      = "dead"
)

type Event struct {
	ID          string    `gorm:"primaryKey;size:32"`
	TenantID    string    `gorm:"index;size:32;not null"`
	EventType   string    `gorm:"size:64;not null"`
	AggregateID string    `gorm:"index;size:64;not null"`
	Payload     []byte    `gorm:"type:json;not null"`
	Status      string    `gorm:"index;size:16;not null"`
	Attempts    int       `gorm:"not null;default:0"`
	AvailableAt time.Time `gorm:"index;not null"`
	CreatedAt   time.Time
	DeliveredAt *time.Time
}

type EventInput struct {
	TenantID, EventType, AggregateID string
	Payload                          []byte
}
