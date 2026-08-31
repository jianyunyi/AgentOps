package auth

import (
	"context"
	"errors"

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
