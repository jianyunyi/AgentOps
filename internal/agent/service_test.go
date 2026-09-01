package agent

import (
	"agentscope/internal/audit"
	"context"
	"errors"
	"strings"
	"testing"
)

type failingAuditRepository struct{}

func (failingAuditRepository) List(context.Context, string, int, int, string) ([]audit.Record, int64, error) {
	return nil, 0, errors.New("audit unavailable")
}

func (failingAuditRepository) Append(context.Context, *audit.Record) error {
	return errors.New("audit unavailable")
}

type fakeRepository struct {
	agent       *Agent
	credential  *AgentCredential
	credentials []*AgentCredential
}

func (f *fakeRepository) CreateAgent(_ context.Context, agent *Agent) error {
	f.agent = agent
	return nil
}

func (f *fakeRepository) FindAgent(_ context.Context, tenantID, agentID string) (*Agent, error) {
	if f.agent == nil || f.agent.TenantID != tenantID || f.agent.ID != agentID {
		return nil, ErrAgentNotFound
	}
	return f.agent, nil
}

func (f *fakeRepository) CreateCredential(_ context.Context, credential *AgentCredential) error {
	if f.credential == nil {
		f.credential = credential
	} else {
		f.credentials = append(f.credentials, f.credential)
		f.credential = credential
	}
	return nil
}

func (f *fakeRepository) FindCredentialByHash(_ context.Context, hash string) (*AgentCredential, error) {
	if f.credential == nil || f.credential.KeyHash != hash {
		return nil, ErrCredentialNotFound
	}
	return f.credential, nil
}

func (f *fakeRepository) ListAgents(_ context.Context, tenantID string) ([]Agent, error) {
	if f.agent != nil && f.agent.TenantID == tenantID {
		return []Agent{*f.agent}, nil
	}
	return []Agent{}, nil
}

func (f *fakeRepository) RevokeCredentials(_ context.Context, tenantID, agentID string) error {
	if f.credential != nil && f.credential.TenantID == tenantID && f.credential.AgentID == agentID {
		f.credential.Status = CredentialRevoked
	}
	for _, credential := range f.credentials {
		if credential.TenantID == tenantID && credential.AgentID == agentID {
			credential.Status = CredentialRevoked
		}
	}
	return nil
}

func TestCreateAgentReturnsRawKeyOnceAndPersistsOnlyHash(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	result, err := svc.CreateAgent(context.Background(), CreateAgentInput{
		TenantID:    "tenant_001",
		Name:        "IT Ops Agent",
		Description: "Internal operations assistant",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	if result.RawAPIKey == "" {
		t.Fatal("CreateAgent() must return a raw API key once")
	}
	if repo.credential == nil || repo.credential.KeyHash == "" {
		t.Fatal("CreateAgent() must persist a key hash")
	}
	if repo.credential.KeyHash == result.RawAPIKey {
		t.Fatal("raw API key must not be persisted")
	}
	if repo.credential.TenantID != "tenant_001" || repo.credential.AgentID != result.Agent.ID {
		t.Fatalf("credential identity mismatch: %+v", repo.credential)
	}
}

func TestAuthenticateAPIKeyResolvesTenantAndAgent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops", Environment: "production"})
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	identity, err := svc.AuthenticateAPIKey(context.Background(), created.RawAPIKey)
	if err != nil {
		t.Fatalf("AuthenticateAPIKey() error = %v", err)
	}
	if identity.TenantID != "tenant_001" || identity.AgentID != created.Agent.ID {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestAuthenticateAPIKeyRejectsUnknownKey(t *testing.T) {
	svc := NewService(&fakeRepository{})
	if _, err := svc.AuthenticateAPIKey(context.Background(), "ag_live_unknown"); err == nil {
		t.Fatal("AuthenticateAPIKey() should reject an unknown key")
	}
}

func TestAuthenticateAPIKeyRejectsSuspendedAgent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops"})
	if err != nil {
		t.Fatal(err)
	}
	repo.agent.Status = AgentStatusSuspended
	if _, err := svc.AuthenticateAPIKey(context.Background(), created.RawAPIKey); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("suspended Agent error = %v, want ErrInvalidAPIKey", err)
	}
}

func TestAuthenticateAPIKeyRejectsMissingAgent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops"})
	if err != nil {
		t.Fatal(err)
	}
	repo.agent = nil
	if _, err := svc.AuthenticateAPIKey(context.Background(), created.RawAPIKey); !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("missing Agent error = %v, want ErrInvalidAPIKey", err)
	}
}

func TestRotateAPIKeyRevokesPreviousCredential(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops", Environment: "production"})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := svc.RotateAPIKey(context.Background(), "tenant_001", created.Agent.ID)
	if err != nil {
		t.Fatalf("RotateAPIKey() error = %v", err)
	}
	if rotated.RawAPIKey == "" || rotated.RawAPIKey == created.RawAPIKey {
		t.Fatal("rotation must return a new raw key")
	}
	if len(repo.credentials) != 1 || repo.credentials[0].Status != CredentialRevoked {
		t.Fatal("rotation must revoke the old credential")
	}
}

func TestRotateAPIKeyRejectsMissingAgent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	if _, err := svc.RotateAPIKey(context.Background(), "tenant_001", "agt_missing"); !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("RotateAPIKey() error = %v, want ErrAgentNotFound", err)
	}
	if repo.credential != nil || len(repo.credentials) != 0 {
		t.Fatal("rotation for a missing Agent must not create credentials")
	}
}

func TestRevokeAPIKeyRejectsCrossTenantAgent(t *testing.T) {
	repo := &fakeRepository{agent: &Agent{ID: "agt_001", TenantID: "tenant_002", Status: AgentStatusActive}}
	svc := NewService(repo)
	if err := svc.RevokeAPIKey(context.Background(), "tenant_001", "agt_001"); !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("RevokeAPIKey() error = %v, want ErrAgentNotFound", err)
	}
}

func TestCreateAgentDoesNotSilentlyIgnoreAuditFailure(t *testing.T) {
	svc := NewServiceWithAudit(&fakeRepository{}, audit.NewService(failingAuditRepository{}))
	_, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops"})
	if err == nil || !strings.Contains(err.Error(), "audit write failed") {
		t.Fatalf("expected explicit audit failure, got %v", err)
	}
}
