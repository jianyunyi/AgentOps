package auth

import (
	"context"
	"errors"

	"agentscope/internal/tenant"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, session *Session) error
	FindSession(ctx context.Context, sessionID string) (*Session, error)
	RegisterTenantOwner(ctx context.Context, tenantName string, user *User) (string, error)
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
