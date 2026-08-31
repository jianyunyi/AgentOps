package audit

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID           uint            `gorm:"primaryKey"`
	TenantID     string          `gorm:"index;size:32;not null"`
	ActorID      string          `gorm:"index;size:32;not null"`
	ActorType    string          `gorm:"size:16;not null"`
	Action       string          `gorm:"size:64;not null"`
	ResourceType string          `gorm:"size:32;not null"`
	ResourceID   string          `gorm:"size:64;not null"`
	Before       json.RawMessage `gorm:"type:json"`
	After        json.RawMessage `gorm:"type:json"`
	RequestID    string          `gorm:"index;size:64"`
	CreatedAt    time.Time
}

type RecordInput struct {
	TenantID     string
	ActorID      string
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   string
	Before       map[string]any
	After        map[string]any
	RequestID    string
}
