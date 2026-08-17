package agent

import "time"

type Agent struct {
	ID          string `gorm:"primaryKey;size:32"`
	TenantID    string `gorm:"index;size:32;not null"`
	Name        string `gorm:"size:128;not null"`
	Description string `gorm:"size:512"`
	Environment string `gorm:"size:32;not null"`
	Status      string `gorm:"size:16;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AgentCredential struct {
	ID        string `gorm:"primaryKey;size:32"`
	TenantID  string `gorm:"uniqueIndex:idx_agent_key_hash;size:32;not null"`
	AgentID   string `gorm:"index;size:32;not null"`
	KeyPrefix string `gorm:"size:24;not null"`
	KeyHash   string `gorm:"uniqueIndex:idx_agent_key_hash;size:64;not null"`
	Status    string `gorm:"size:16;not null"`
	CreatedAt time.Time
	RevokedAt *time.Time
}

const (
	AgentStatusActive    = "active"
	AgentStatusSuspended = "suspended"
	CredentialActive     = "active"
	CredentialRevoked    = "revoked"
)
