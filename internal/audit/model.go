package audit

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	TenantID     string          `gorm:"index;size:32;not null" json:"tenant_id"`
	ActorID      string          `gorm:"index;size:32;not null" json:"actor_id"`
	ActorType    string          `gorm:"size:16;not null" json:"actor_type"`
	Action       string          `gorm:"size:64;not null" json:"action"`
	ResourceType string          `gorm:"size:32;not null" json:"resource_type"`
	ResourceID   string          `gorm:"size:64;not null" json:"resource_id"`
	Before       json.RawMessage `gorm:"type:json" json:"before"`
	After        json.RawMessage `gorm:"type:json" json:"after"`
	RequestID    string          `gorm:"index;size:64" json:"request_id"`
	CreatedAt    time.Time       `json:"created_at"`
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
