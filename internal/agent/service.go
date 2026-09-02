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
	Agent            Agent
	RawAPIKey        string
	RawSigningSecret string
}

type Identity struct {
	TenantID string
	AgentID  string
}

type Service struct {
	repo              Repository
	audit             *audit.Service
	nonceStore        NonceStore
	replayWindow      time.Duration
	nonceTTL          time.Duration
	signingProtector  SigningSecretProtector
	signingEnabled    bool
	signatureRequired bool
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithAudit(repo Repository, auditService *audit.Service) *Service {
	return &Service{repo: repo, audit: auditService}
}

func NewServiceWithNonceStore(repo Repository, nonceStore NonceStore, replayWindow, nonceTTL time.Duration) *Service {
	return &Service{repo: repo, nonceStore: nonceStore, replayWindow: normalizeReplayWindow(replayWindow), nonceTTL: normalizeNonceTTL(nonceTTL)}
}

func NewServiceWithAuditAndNonceStore(repo Repository, auditService *audit.Service, nonceStore NonceStore, replayWindow, nonceTTL time.Duration) *Service {
	return &Service{repo: repo, audit: auditService, nonceStore: nonceStore, replayWindow: normalizeReplayWindow(replayWindow), nonceTTL: normalizeNonceTTL(nonceTTL)}
}

func NewServiceWithAuditAndNonceStoreAndSigning(repo Repository, auditService *audit.Service, nonceStore NonceStore, replayWindow, nonceTTL time.Duration, protector SigningSecretProtector, signatureRequired bool) *Service {
	return &Service{repo: repo, audit: auditService, nonceStore: nonceStore, replayWindow: normalizeReplayWindow(replayWindow), nonceTTL: normalizeNonceTTL(nonceTTL), signingProtector: protector, signingEnabled: true, signatureRequired: signatureRequired}
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
	rawSigningSecret, err := s.prepareSigningSecret(&credential)
	if err != nil {
		return CreateAgentResult{}, err
	}
	if s.audit != nil {
		input := audit.RecordInput{TenantID: agent.TenantID, ActorID: "system", Action: "agent.create", ResourceType: "agent", ResourceID: agent.ID, After: map[string]any{"credential_id": credential.ID, "key_prefix": credential.KeyPrefix, "signing_protocol": signingProtocol(rawSigningSecret)}}
		if atomic, ok := s.repo.(interface {
			CreateAgentWithCredentialAndAudit(context.Context, *Agent, *AgentCredential, *audit.Record) error
		}); ok {
			record, err := s.audit.Prepare(input)
			if err != nil {
				return CreateAgentResult{}, err
			}
			if err := atomic.CreateAgentWithCredentialAndAudit(ctx, &agent, &credential, record); err != nil {
				return CreateAgentResult{}, err
			}
		} else {
			if err := s.repo.CreateAgent(ctx, &agent); err != nil {
				return CreateAgentResult{}, err
			}
			if err := s.repo.CreateCredential(ctx, &credential); err != nil {
				return CreateAgentResult{}, err
			}
			if err := s.audit.Record(ctx, input); err != nil {
				return CreateAgentResult{}, fmt.Errorf("agent created but audit write failed: %w", err)
			}
		}
	} else {
		if err := s.repo.CreateAgent(ctx, &agent); err != nil {
			return CreateAgentResult{}, err
		}
		if err := s.repo.CreateCredential(ctx, &credential); err != nil {
			return CreateAgentResult{}, err
		}
	}
	return CreateAgentResult{Agent: agent, RawAPIKey: rawKey, RawSigningSecret: rawSigningSecret}, nil
}

func (s *Service) AuthenticateAPIKey(ctx context.Context, rawKey string) (Identity, error) {
	credential, identity, err := s.authenticateCredential(ctx, rawKey)
	if err != nil {
		return Identity{}, err
	}
	if toucher, ok := s.repo.(interface {
		TouchCredential(context.Context, string, string) error
	}); ok {
		if err := toucher.TouchCredential(ctx, credential.ID, ""); err != nil {
			return Identity{}, fmt.Errorf("record credential usage: %w", err)
		}
	}
	return identity, nil
}

func (s *Service) AuthenticateIngestRequest(ctx context.Context, rawKey string, metadata AuthenticationMetadata) (Identity, error) {
	if !validAuthenticationMetadata(metadata, normalizeReplayWindow(s.replayWindow)) {
		return Identity{}, ErrInvalidAgentRequest
	}
	credential, identity, err := s.authenticateCredential(ctx, rawKey)
	if err != nil {
		return Identity{}, err
	}
	if err := s.verifyRequestSignature(credential, metadata); err != nil {
		return Identity{}, err
	}
	if s.nonceStore == nil {
		return Identity{}, ErrNonceStoreUnavailable
	}
	claimed, err := s.nonceStore.Claim(ctx, identity.TenantID, identity.AgentID, metadata.Nonce, normalizeNonceTTL(s.nonceTTL))
	if err != nil {
		return Identity{}, ErrNonceStoreUnavailable
	}
	if !claimed {
		return Identity{}, ErrReplayDetected
	}
	if toucher, ok := s.repo.(interface {
		TouchCredential(context.Context, string, string) error
	}); ok {
		if err := toucher.TouchCredential(ctx, credential.ID, ""); err != nil {
			return Identity{}, fmt.Errorf("record credential usage: %w", err)
		}
	}
	return identity, nil
}

func (s *Service) authenticateCredential(ctx context.Context, rawKey string) (*AgentCredential, Identity, error) {
	if !strings.HasPrefix(rawKey, "ag_live_") {
		return nil, Identity{}, ErrInvalidAPIKey
	}
	keyHash := hashKey(rawKey)
	credential, err := s.repo.FindCredentialByHash(ctx, keyHash)
	if err != nil || credential == nil || credential.Status != CredentialActive || subtle.ConstantTimeCompare([]byte(credential.KeyHash), []byte(keyHash)) != 1 {
		return nil, Identity{}, ErrInvalidAPIKey
	}
	agentRecord, err := s.repo.FindAgent(ctx, credential.TenantID, credential.AgentID)
	if err != nil || agentRecord == nil || agentRecord.Status != AgentStatusActive {
		return nil, Identity{}, ErrInvalidAPIKey
	}
	return credential, Identity{TenantID: credential.TenantID, AgentID: credential.AgentID}, nil
}

func (s *Service) verifyRequestSignature(credential *AgentCredential, metadata AuthenticationMetadata) error {
	if len(credential.SigningSecretCiphertext) == 0 {
		if s.signatureRequired {
			return ErrSignatureRequired
		}
		return nil
	}
	if s.signingProtector == nil {
		return ErrSigningSecretUnavailable
	}
	secret, err := s.signingProtector.Decrypt(credential.SigningSecretCiphertext)
	if err != nil {
		return ErrSigningSecretUnavailable
	}
	if err := VerifyAgentSignature(secret, metadata); err != nil {
		return err
	}
	return nil
}

func (s *Service) prepareSigningSecret(credential *AgentCredential) (string, error) {
	if !s.signingEnabled {
		return "", nil
	}
	if s.signingProtector == nil {
		return "", ErrSigningSecretUnavailable
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	ciphertext, err := s.signingProtector.Encrypt(secret)
	if err != nil {
		return "", ErrSigningSecretUnavailable
	}
	credential.SigningSecretCiphertext = ciphertext
	return encodeSigningSecret(secret), nil
}

func signingProtocol(rawSecret string) string {
	if rawSecret == "" {
		return "legacy"
	}
	return signingVersion
}

func validAuthenticationMetadata(metadata AuthenticationMetadata, replayWindow time.Duration) bool {
	if metadata.Timestamp <= 0 || replayWindow <= 0 {
		return false
	}
	if delta := time.Now().Unix() - metadata.Timestamp; delta > int64(replayWindow/time.Second) || delta < -int64(replayWindow/time.Second) {
		return false
	}
	if len(metadata.Nonce) < 1 || len(metadata.Nonce) > 128 {
		return false
	}
	for index := 0; index < len(metadata.Nonce); index++ {
		if metadata.Nonce[index] < 0x21 || metadata.Nonce[index] > 0x7e {
			return false
		}
	}
	return true
}

func normalizeReplayWindow(value time.Duration) time.Duration {
	if value <= 0 {
		return 5 * time.Minute
	}
	return value
}

func normalizeNonceTTL(value time.Duration) time.Duration {
	if value <= 0 {
		return 10 * time.Minute
	}
	return value
}

func (s *Service) RotateAPIKey(ctx context.Context, tenantID, agentID string) (CreateAgentResult, error) {
	if tenantID == "" || agentID == "" {
		return CreateAgentResult{}, errors.New("tenant and agent are required")
	}
	if target, err := s.repo.FindAgent(ctx, tenantID, agentID); err != nil || target == nil {
		return CreateAgentResult{}, ErrAgentNotFound
	}
	rawKey, err := randomKey()
	if err != nil {
		return CreateAgentResult{}, err
	}
	credential := &AgentCredential{ID: mustRandomID("cred_"), TenantID: tenantID, AgentID: agentID, KeyPrefix: rawKey[:min(16, len(rawKey))], KeyHash: hashKey(rawKey), Status: CredentialActive, CreatedAt: time.Now().UTC()}
	rawSigningSecret, err := s.prepareSigningSecret(credential)
	if err != nil {
		return CreateAgentResult{}, err
	}
	if s.audit != nil {
		input := audit.RecordInput{TenantID: tenantID, ActorID: "system", Action: "agent.key.rotate", ResourceType: "agent", ResourceID: agentID, After: map[string]any{"credential_id": credential.ID, "key_prefix": credential.KeyPrefix, "signing_protocol": signingProtocol(rawSigningSecret)}}
		if atomic, ok := s.repo.(interface {
			RotateCredentialWithAudit(context.Context, string, string, *AgentCredential, *audit.Record) error
		}); ok {
			record, err := s.audit.Prepare(input)
			if err != nil {
				return CreateAgentResult{}, err
			}
			if err := atomic.RotateCredentialWithAudit(ctx, tenantID, agentID, credential, record); err != nil {
				return CreateAgentResult{}, err
			}
		} else {
			if err := s.repo.RevokeCredentials(ctx, tenantID, agentID); err != nil {
				return CreateAgentResult{}, err
			}
			if err := s.repo.CreateCredential(ctx, credential); err != nil {
				return CreateAgentResult{}, err
			}
			if err := s.audit.Record(ctx, input); err != nil {
				return CreateAgentResult{}, fmt.Errorf("key rotated but audit write failed: %w", err)
			}
		}
	} else {
		if err := s.repo.RevokeCredentials(ctx, tenantID, agentID); err != nil {
			return CreateAgentResult{}, err
		}
		if err := s.repo.CreateCredential(ctx, credential); err != nil {
			return CreateAgentResult{}, err
		}
	}
	return CreateAgentResult{Agent: Agent{ID: agentID, TenantID: tenantID}, RawAPIKey: rawKey, RawSigningSecret: rawSigningSecret}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, tenantID, agentID string) error {
	if tenantID == "" || agentID == "" {
		return errors.New("tenant and agent are required")
	}
	if target, err := s.repo.FindAgent(ctx, tenantID, agentID); err != nil || target == nil {
		return ErrAgentNotFound
	}
	if s.audit != nil {
		input := audit.RecordInput{TenantID: tenantID, ActorID: "system", Action: "agent.key.revoke", ResourceType: "agent", ResourceID: agentID}
		if atomic, ok := s.repo.(interface {
			RevokeCredentialWithAudit(context.Context, string, string, *audit.Record) error
		}); ok {
			record, err := s.audit.Prepare(input)
			if err != nil {
				return err
			}
			return atomic.RevokeCredentialWithAudit(ctx, tenantID, agentID, record)
		}
		if err := s.repo.RevokeCredentials(ctx, tenantID, agentID); err != nil {
			return err
		}
		if err := s.audit.Record(ctx, input); err != nil {
			return fmt.Errorf("key revoked but audit write failed: %w", err)
		}
		return nil
	}
	return s.repo.RevokeCredentials(ctx, tenantID, agentID)
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
