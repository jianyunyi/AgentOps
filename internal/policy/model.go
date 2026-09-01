package policy

import "time"

type Policy struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	TenantID      string    `gorm:"index;size:32;not null" json:"tenant_id"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	Version       int       `gorm:"not null" json:"version"`
	Enabled       bool      `gorm:"index;not null" json:"enabled"`
	RulesEnabled  bool      `gorm:"not null;default:true" json:"rules_enabled"`
	LLMEnabled    bool      `gorm:"not null;default:false" json:"llm_enabled"`
	MaxInputBytes int       `gorm:"not null;default:65536" json:"max_input_bytes"`
	CreatedBy     string    `gorm:"size:32;not null" json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
