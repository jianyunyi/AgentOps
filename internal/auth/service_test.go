package auth

import (
	"context"
	"testing"
)

type fakeUserRepository struct {
	users    map[string]*User
	sessions map[string]*Session
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, email string) (*User, error) {
	user, ok := f.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (f *fakeUserRepository) CreateSession(_ context.Context, session *Session) error {
	if f.sessions == nil {
		f.sessions = map[string]*Session{}
	}
	f.sessions[session.ID] = session
	return nil
}

func (f *fakeUserRepository) FindSession(_ context.Context, sessionID string) (*Session, error) {
	session, ok := f.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func TestLoginCreatesTenantScopedSession(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	repo := &fakeUserRepository{users: map[string]*User{
		"owner@example.com": {
			ID: "usr_001", TenantID: "tenant_001", Email: "owner@example.com", PasswordHash: hash, Role: RoleOwner, Status: UserActive,
		},
	}}
	svc := NewService(repo, "test-session-secret")

	session, err := svc.Login(context.Background(), "owner@example.com", "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if session.ID == "" || session.TenantID != "tenant_001" || session.UserID != "usr_001" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if repo.sessions[session.ID] == nil {
		t.Fatal("Login() must persist the session")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")
	repo := &fakeUserRepository{users: map[string]*User{
		"owner@example.com": {ID: "usr_001", TenantID: "tenant_001", Email: "owner@example.com", PasswordHash: hash, Role: RoleOwner, Status: UserActive},
	}}
	svc := NewService(repo, "test-session-secret")
	if _, err := svc.Login(context.Background(), "owner@example.com", "wrong-password"); err != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestPermissionRequiresRoleCapability(t *testing.T) {
	if !HasPermission(RoleAuditor, PermissionRiskReview) {
		t.Fatal("auditor must be allowed to review risks")
	}
	if HasPermission(RoleViewer, PermissionRiskReview) {
		t.Fatal("viewer must not be allowed to review risks")
	}
}
