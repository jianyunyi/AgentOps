package agent

import "time"

type Agent struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	TenantID    string    `gorm:"index;size:32;not null" json:"tenantId"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	Environment string    `gorm:"size:32;not null" json:"environment"`
	Status      string    `gorm:"size:16;not null" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AgentCredential struct {
	ID                      string `gorm:"primaryKey;size:32"`
	TenantID                string `gorm:"uniqueIndex:idx_agent_key_hash;size:32;not null"`
	AgentID                 string `gorm:"index;size:32;not null"`
	KeyPrefix               string `gorm:"size:24;not null"`
	KeyHash                 string `gorm:"uniqueIndex:idx_agent_key_hash;size:64;not null"`
	SigningSecretCiphertext []byte `gorm:"column:signing_secret_ciphertext;type:blob" json:"-"`
	Status                  string `gorm:"size:16;not null"`
	CreatedAt               time.Time
	RevokedAt               *time.Time
	ExpiresAt               *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt              *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedIP              string     `gorm:"size:64" json:"lastUsedIp,omitempty"`
}

const (
	AgentStatusActive    = "active"
	AgentStatusSuspended = "suspended"
	CredentialActive     = "active"
	CredentialRevoked    = "revoked"
)
