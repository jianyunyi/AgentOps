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

func (f *fakeUserRepository) ListMembers(_ context.Context, tenantID string) ([]User, error) {
	var result []User
	for _, user := range f.users {
		if user.TenantID == tenantID {
			copy := *user
			result = append(result, copy)
		}
	}
	return result, nil
}

func (f *fakeUserRepository) UpdateMemberRole(_ context.Context, tenantID, memberID, role string) error {
	for _, user := range f.users {
		if user.ID == memberID && user.TenantID == tenantID {
			user.Role = role
			return nil
		}
	}
	return ErrUserNotFound
}

func (f *fakeUserRepository) DisableMember(_ context.Context, tenantID, memberID string) error {
	for _, user := range f.users {
		if user.ID == memberID && user.TenantID == tenantID {
			user.Status = UserDisabled
			return nil
		}
	}
	return ErrUserNotFound
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
	if !HasPermission(RoleAdmin, PermissionMemberRead) || !HasPermission(RoleAdmin, PermissionMemberWrite) {
		t.Fatal("admin must be allowed to manage members")
	}
}

func TestMemberServiceValidatesOwnerBoundaries(t *testing.T) {
	repo := &fakeUserRepository{users: map[string]*User{
		"owner@example.com": {ID: "usr_owner", TenantID: "ten_001", Role: RoleOwner, Status: UserActive},
		"dev@example.com":   {ID: "usr_dev", TenantID: "ten_001", Role: RoleDeveloper, Status: UserActive},
	}}
	svc := NewService(repo, "secret")
	if err := svc.ChangeMemberRole(context.Background(), "ten_001", "usr_owner", "usr_owner", RoleViewer); err == nil {
		t.Fatal("must not change owner role")
	}
	if err := svc.DisableMember(context.Background(), "ten_001", "usr_owner", "usr_owner"); err == nil {
		t.Fatal("must not disable self")
	}
	if err := svc.ChangeMemberRole(context.Background(), "ten_001", "usr_owner", "usr_dev", RoleAuditor); err != nil {
		t.Fatalf("ChangeMemberRole() error = %v", err)
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
