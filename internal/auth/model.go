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
	ID           string  `gorm:"primaryKey;size:32"`
	TenantID     string  `gorm:"index;size:32;not null"`
	Email        string  `gorm:"uniqueIndex;size:254;not null"`
	PasswordHash string  `gorm:"size:128;not null"`
	OIDCIssuer   *string `gorm:"size:512;uniqueIndex:ux_user_oidc" json:"-"`
	OIDCSubject  *string `gorm:"size:255;uniqueIndex:ux_user_oidc" json:"-"`
	Role         string  `gorm:"size:32;not null"`
	Status       string  `gorm:"size:16;not null"`
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

const (
	InvitePending  = "pending"
	InviteAccepted = "accepted"
	InviteRevoked  = "revoked"
	InviteExpired  = "expired"
)

type MemberInvitation struct {
	ID         string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID   string     `gorm:"index;size:32;not null" json:"tenant_id"`
	Email      string     `gorm:"index;size:254;not null" json:"email"`
	Role       string     `gorm:"size:32;not null" json:"role"`
	TokenHash  string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	InvitedBy  string     `gorm:"size:32;not null" json:"invited_by"`
	Status     string     `gorm:"index;size:16;not null" json:"status"`
	ExpiresAt  time.Time  `gorm:"index;not null" json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
