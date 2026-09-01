package auth

import (
	"context"
	"errors"
	"strings"

	"agentscope/internal/audit"
	"agentscope/internal/tenant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationUsed     = errors.New("invitation is no longer usable")
)

type MemberFilter struct {
	Query  string
	Status string
	Role   string
	Offset int
	Limit  int
}

type InvitationRepository interface {
	CreateInvitation(context.Context, *MemberInvitation) error
	ListInvitations(context.Context, string, int, int) ([]MemberInvitation, int64, error)
	FindInvitationByTokenHash(context.Context, string) (*MemberInvitation, error)
	AcceptInvitation(context.Context, *MemberInvitation, *User, *Session, *audit.Record) error
}

type InvitationMutationRepository interface {
	CreateInvitationWithAudit(context.Context, *MemberInvitation, *audit.Record) error
}

type MemberQueryRepository interface {
	ListMembersFiltered(context.Context, string, MemberFilter) ([]User, int64, error)
}

type OIDCRepository interface {
	FindByOIDC(context.Context, string, string) (*User, error)
	BindOIDC(context.Context, string, string, string) error
	CreateOIDCUser(context.Context, *User) error
}

type MemberMutationRepository interface {
	UpdateMemberRoleWithAudit(context.Context, string, string, string, *audit.Record) error
	DisableMemberWithAudit(context.Context, string, string, *audit.Record) error
	TransferOwnerWithAudit(context.Context, string, string, string, *audit.Record) error
}

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, session *Session) error
	FindSession(ctx context.Context, sessionID string) (*Session, error)
	FindUserByID(ctx context.Context, userID string) (*User, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RegisterTenantOwner(ctx context.Context, tenantName string, user *User) (string, error)
	ListMembers(ctx context.Context, tenantID string) ([]User, error)
	UpdateMemberRole(ctx context.Context, tenantID, memberID, role string) error
	DisableMember(ctx context.Context, tenantID, memberID string) error
}

func (r *GORMRepository) ListMembers(ctx context.Context, tenantID string) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).Select("id", "tenant_id", "email", "role", "status", "created_at", "updated_at").Where("tenant_id = ?", tenantID).Order("created_at asc").Find(&users).Error
	return users, err
}

func (r *GORMRepository) ListMembersFiltered(ctx context.Context, tenantID string, filter MemberFilter) ([]User, int64, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		query = query.Where("LOWER(email) LIKE ?", like)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	var total int64
	if err := query.Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := query.Select("id", "tenant_id", "email", "role", "status", "created_at", "updated_at").Order("created_at asc").Offset(filter.Offset).Limit(filter.Limit).Find(&users).Error
	return users, total, err
}

func (r *GORMRepository) UpdateMemberRole(ctx context.Context, tenantID, memberID, role string) error {
	result := r.db.WithContext(ctx).Model(&User{}).Where("id = ? AND tenant_id = ? AND status = ? AND role <> ?", memberID, tenantID, UserActive, RoleOwner).Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *GORMRepository) DisableMember(ctx context.Context, tenantID, memberID string) error {
	result := r.db.WithContext(ctx).Model(&User{}).Where("id = ? AND tenant_id = ? AND status = ? AND role <> ?", memberID, tenantID, UserActive, RoleOwner).Update("status", UserDisabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *GORMRepository) UpdateMemberRoleWithAudit(ctx context.Context, tenantID, memberID, role string, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).Where("id = ? AND tenant_id = ? AND status = ? AND role <> ?", memberID, tenantID, UserActive, RoleOwner).Update("role", role)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) DisableMemberWithAudit(ctx context.Context, tenantID, memberID string, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).Where("id = ? AND tenant_id = ? AND status = ? AND role <> ?", memberID, tenantID, UserActive, RoleOwner).Update("status", UserDisabled)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) TransferOwnerWithAudit(ctx context.Context, tenantID, currentOwnerID, targetID string, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ? AND status = ?", targetID, tenantID, UserActive).First(&target).Error; err != nil {
			return ErrUserNotFound
		}
		if target.Role == RoleOwner {
			return ErrInvalidMemberOperation
		}
		result := tx.Model(&User{}).Where("id = ? AND tenant_id = ? AND role = ? AND status = ?", currentOwnerID, tenantID, RoleOwner, UserActive).Update("role", RoleAdmin)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		if err := tx.Model(&User{}).Where("id = ?", targetID).Update("role", RoleOwner).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) CreateInvitation(ctx context.Context, invitation *MemberInvitation) error {
	return r.db.WithContext(ctx).Create(invitation).Error
}

func (r *GORMRepository) CreateInvitationWithAudit(ctx context.Context, invitation *MemberInvitation, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invitation).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) ListInvitations(ctx context.Context, tenantID string, offset, limit int) ([]MemberInvitation, int64, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	var total int64
	if err := query.Model(&MemberInvitation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var invitations []MemberInvitation
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&invitations).Error
	return invitations, total, err
}

func (r *GORMRepository) FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*MemberInvitation, error) {
	var invitation MemberInvitation
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

func (r *GORMRepository) AcceptInvitation(ctx context.Context, invitation *MemberInvitation, user *User, session *Session, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current MemberInvitation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", invitation.ID).First(&current).Error; err != nil {
			return err
		}
		if current.Status != InvitePending || !current.ExpiresAt.After(timeNow()) {
			return ErrInvitationUsed
		}
		user.TenantID = current.TenantID
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Model(&current).Updates(map[string]any{"status": InviteAccepted, "accepted_at": timeNow()}).Error; err != nil {
			return err
		}
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *GORMRepository) FindByOIDC(ctx context.Context, issuer, subject string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("oidc_issuer = ? AND oidc_subject = ?", issuer, subject).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *GORMRepository) BindOIDC(ctx context.Context, userID, issuer, subject string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]any{"oidc_issuer": issuer, "oidc_subject": subject}).Error
}

func (r *GORMRepository) CreateOIDCUser(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GORMRepository) CreateSession(ctx context.Context, session *Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *GORMRepository) FindSession(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	if err := r.db.WithContext(ctx).Where("id = ? AND expires_at > ?", sessionID, timeNow()).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (r *GORMRepository) FindUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Select("id", "tenant_id", "role", "status").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *GORMRepository) RevokeSession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Where("id = ?", sessionID).Delete(&Session{}).Error
}

func (r *GORMRepository) RegisterTenantOwner(ctx context.Context, tenantName string, user *User) (string, error) {
	tenantRecord := tenant.Tenant{ID: "ten_" + user.ID[4:], Name: tenantName, Status: "active", CreatedAt: timeNow(), UpdatedAt: timeNow()}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenantRecord).Error; err != nil {
			return err
		}
		user.TenantID = tenantRecord.ID
		return tx.Create(user).Error
	})
	if err != nil {
		return "", err
	}
	return tenantRecord.ID, nil
}
