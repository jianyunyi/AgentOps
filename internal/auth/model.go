package auth

import "time"

const (
	UserActive   = "active"
	UserDisabled = "disabled"

	RoleOwner     = "owner"
	RoleAdmin     = "admin"
	RoleDeveloper = "developer"
	RoleAuditor   = "auditor"
	RoleViewer    = "viewer"
)

const (
	PermissionAgentRead   = "agent:read"
	PermissionAgentWrite  = "agent:write"
	PermissionRiskRead    = "risk:read"
	PermissionRiskReview  = "risk:review"
	PermissionAuditRead   = "audit:read"
	PermissionMemberRead  = "member:read"
	PermissionMemberWrite = "member:write"
)

type User struct {
	ID           string `gorm:"primaryKey;size:32"`
	TenantID     string `gorm:"index;size:32;not null"`
	Email        string `gorm:"uniqueIndex;size:254;not null"`
	PasswordHash string `gorm:"size:128;not null"`
	Role         string `gorm:"size:32;not null"`
	Status       string `gorm:"size:16;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    string    `gorm:"index;size:32;not null"`
	TenantID  string    `gorm:"index;size:32;not null"`
	Role      string    `gorm:"size:32;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}
