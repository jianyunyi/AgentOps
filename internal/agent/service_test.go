package agent

import (
	"context"
	"testing"
)

type fakeRepository struct {
	agent      *Agent
	credential *AgentCredential
}

func (f *fakeRepository) CreateAgent(_ context.Context, agent *Agent) error {
	f.agent = agent
	return nil
}

func (f *fakeRepository) CreateCredential(_ context.Context, credential *AgentCredential) error {
	f.credential = credential
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
