package auth

import (
	"context"
	"testing"
)

type fakeUserRepository struct {
	users              map[string]*User
	sessions           map[string]*Session
	registeredTenantID string
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

func (f *fakeUserRepository) RegisterTenantOwner(_ context.Context, tenantName string, user *User) (string, error) {
	if tenantName == "" || user == nil {
		return "", ErrInvalidRegistration
	}
	f.registeredTenantID = "ten_001"
	if f.users == nil {
		f.users = map[string]*User{}
	}
	f.users[user.Email] = user
	return f.registeredTenantID, nil
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

func TestRegisterCreatesTenantOwnerAndSession(t *testing.T) {
	repo := &fakeUserRepository{users: map[string]*User{}}
	svc := NewService(repo, "test-session-secret")

	session, err := svc.Register(context.Background(), "Acme Operations", "owner@example.com", "correct-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if session.TenantID != "ten_001" || session.Role != RoleOwner {
		t.Fatalf("unexpected session: %+v", session)
	}
	if repo.users["owner@example.com"] == nil {
		t.Fatal("registration must create owner user")
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	if _, err := NewService(&fakeUserRepository{}, "secret").Register(context.Background(), "Acme", "owner@example.com", "short"); err == nil {
		t.Fatal("Register() must reject a short password")
	}
}
