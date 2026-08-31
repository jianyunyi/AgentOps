package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	repo          Repository
	sessionSecret string
}

func NewService(repo Repository, sessionSecret string) *Service {
	return &Service{repo: repo, sessionSecret: sessionSecret}
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	user, err := s.repo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || user == nil || user.Status != UserActive {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	session := &Session{
		ID: sessionID, UserID: user.ID, TenantID: user.TenantID, Role: user.Role,
		ExpiresAt: time.Now().UTC().Add(8 * time.Hour), CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ResolveSession(ctx context.Context, sessionID string) (*Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}
	return s.repo.FindSession(ctx, sessionID)
}

func HasPermission(role, permission string) bool {
	if role == RoleOwner {
		return true
	}
	permissions := map[string]map[string]bool{
		RoleAdmin:     {PermissionAgentRead: true, PermissionAgentWrite: true, PermissionRiskRead: true},
		RoleDeveloper: {PermissionAgentRead: true, PermissionRiskRead: true},
		RoleAuditor:   {PermissionRiskRead: true, PermissionRiskReview: true, PermissionAuditRead: true},
		RoleViewer:    {PermissionAgentRead: true, PermissionRiskRead: true},
	}
	return permissions[role][permission]
}

func randomSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "ses_" + hex.EncodeToString(buf), nil
}

func timeNow() time.Time { return time.Now().UTC() }
