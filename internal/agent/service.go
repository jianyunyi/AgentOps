package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"agentscope/internal/audit"
)

var ErrInvalidAPIKey = errors.New("invalid agent api key")

type CreateAgentInput struct {
	TenantID    string
	Name        string
	Description string
	Environment string
}

type CreateAgentResult struct {
	Agent     Agent
	RawAPIKey string
}

type Identity struct {
	TenantID string
	AgentID  string
}

type Service struct {
	repo  Repository
	audit *audit.Service
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithAudit(repo Repository, auditService *audit.Service) *Service {
	return &Service{repo: repo, audit: auditService}
}

func (s *Service) CreateAgent(ctx context.Context, input CreateAgentInput) (CreateAgentResult, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Name) == "" {
		return CreateAgentResult{}, errors.New("tenant id and agent name are required")
	}
	if input.Environment == "" {
		input.Environment = "development"
	}

	agentID, err := randomID("agt_")
	if err != nil {
		return CreateAgentResult{}, err
	}
	rawKey, err := randomKey()
	if err != nil {
		return CreateAgentResult{}, err
	}
	agent := Agent{
		ID:          agentID,
		TenantID:    input.TenantID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Environment: input.Environment,
		Status:      AgentStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	credential := AgentCredential{
		ID:        mustRandomID("cred_"),
		TenantID:  agent.TenantID,
		AgentID:   agent.ID,
		KeyPrefix: rawKey[:min(16, len(rawKey))],
		KeyHash:   hashKey(rawKey),
		Status:    CredentialActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.CreateAgent(ctx, &agent); err != nil {
		return CreateAgentResult{}, err
	}
	if err := s.repo.CreateCredential(ctx, &credential); err != nil {
		return CreateAgentResult{}, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.RecordInput{TenantID: agent.TenantID, ActorID: "system", Action: "agent.create", ResourceType: "agent", ResourceID: agent.ID, After: map[string]any{"name": agent.Name, "environment": agent.Environment}})
	}
	return CreateAgentResult{Agent: agent, RawAPIKey: rawKey}, nil
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, rawKey string) (Identity, error) {
	if !strings.HasPrefix(rawKey, "ag_live_") {
		return Identity{}, ErrInvalidAPIKey
	}
	credential, err := s.repo.FindCredentialByHash(ctx, hashKey(rawKey))
	if err != nil || credential == nil || subtle.ConstantTimeCompare([]byte(credential.KeyHash), []byte(hashKey(rawKey))) != 1 {
		return Identity{}, ErrInvalidAPIKey
	}
	return Identity{TenantID: credential.TenantID, AgentID: credential.AgentID}, nil
}

func (s *Service) RotateAPIKey(ctx context.Context, tenantID, agentID string) (CreateAgentResult, error) {
	if tenantID == "" || agentID == "" {
		return CreateAgentResult{}, errors.New("tenant and agent are required")
	}
	if err := s.repo.RevokeCredentials(ctx, tenantID, agentID); err != nil {
		return CreateAgentResult{}, err
	}
	rawKey, err := randomKey()
	if err != nil {
		return CreateAgentResult{}, err
	}
	credential := &AgentCredential{ID: mustRandomID("cred_"), TenantID: tenantID, AgentID: agentID, KeyPrefix: rawKey[:min(16, len(rawKey))], KeyHash: hashKey(rawKey), Status: CredentialActive, CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateCredential(ctx, credential); err != nil {
		return CreateAgentResult{}, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.RecordInput{TenantID: tenantID, ActorID: "system", Action: "agent.key.rotate", ResourceType: "agent", ResourceID: agentID, After: map[string]any{"key_prefix": credential.KeyPrefix}})
	}
	return CreateAgentResult{Agent: Agent{ID: agentID, TenantID: tenantID}, RawAPIKey: rawKey}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, tenantID, agentID string) error {
	if tenantID == "" || agentID == "" {
		return errors.New("tenant and agent are required")
	}
	err := s.repo.RevokeCredentials(ctx, tenantID, agentID)
	if err == nil && s.audit != nil {
		_ = s.audit.Record(ctx, audit.RecordInput{TenantID: tenantID, ActorID: "system", Action: "agent.key.revoke", ResourceType: "agent", ResourceID: agentID})
	}
	return err
}

func randomKey() (string, error) {
	return randomToken("ag_live_", 24)
}

func randomID(prefix string) (string, error) {
	return randomToken(prefix, 12)
}

func mustRandomID(prefix string) string {
	id, err := randomID(prefix)
	if err != nil {
		panic(fmt.Sprintf("generate %s: %v", prefix, err))
	}
	return id
}

func randomToken(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

func hashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
