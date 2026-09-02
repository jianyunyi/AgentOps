package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAESGCMProtectorRoundTripAndKeyParsing(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	protector, err := NewAESGCMProtector(key)
	if err != nil {
		t.Fatalf("NewAESGCMProtector() error = %v", err)
	}
	ciphertext, err := protector.Encrypt([]byte("signing-secret"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	plaintext, err := protector.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if string(plaintext) != "signing-secret" {
		t.Fatalf("plaintext = %q", plaintext)
	}
	if _, err := NewAESGCMProtectorFromString("not-a-key"); err == nil {
		t.Fatal("invalid encryption key must be rejected")
	}
	if _, err := NewAESGCMProtectorFromString(base64.StdEncoding.EncodeToString(key)); err != nil {
		t.Fatalf("base64 encryption key rejected: %v", err)
	}
}

func TestBuildAndVerifyAgentSignatureBindsRequest(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	body := []byte(`{"event_id":"evt_1"}`)
	timestamp := time.Now().Unix()
	nonce := "nonce-1"
	signature := BuildAgentSignature(secret, "POST", "/api/v1/ingest/events", body, timestamp, nonce)
	metadata := AuthenticationMetadata{Timestamp: timestamp, Nonce: nonce, Signature: signature, Method: "POST", Path: "/api/v1/ingest/events", BodyHash: HashRequestBody(body)}
	if err := VerifyAgentSignature(secret, metadata); err != nil {
		t.Fatalf("VerifyAgentSignature() error = %v", err)
	}
	metadata.BodyHash = HashRequestBody([]byte(`{"event_id":"evt_2"}`))
	if err := VerifyAgentSignature(secret, metadata); !errors.Is(err, ErrInvalidAgentSignature) {
		t.Fatalf("tampered body error = %v, want ErrInvalidAgentSignature", err)
	}
	if !strings.HasPrefix(signature, "v1=") {
		t.Fatalf("signature = %q, want v1 prefix", signature)
	}
}

type signingRepository struct {
	agent      *Agent
	credential *AgentCredential
}

func (r *signingRepository) CreateAgent(_ context.Context, value *Agent) error {
	r.agent = value
	return nil
}

func (r *signingRepository) CreateCredential(_ context.Context, value *AgentCredential) error {
	r.credential = value
	return nil
}

func (r *signingRepository) FindCredentialByHash(_ context.Context, hash string) (*AgentCredential, error) {
	if r.credential == nil || r.credential.KeyHash != hash || r.credential.Status != CredentialActive {
		return nil, ErrCredentialNotFound
	}
	return r.credential, nil
}

func (r *signingRepository) FindAgent(_ context.Context, tenantID, agentID string) (*Agent, error) {
	if r.agent == nil || r.agent.TenantID != tenantID || r.agent.ID != agentID {
		return nil, ErrAgentNotFound
	}
	return r.agent, nil
}

func (r *signingRepository) ListAgents(_ context.Context, tenantID string) ([]Agent, error) {
	if r.agent == nil || r.agent.TenantID != tenantID {
		return []Agent{}, nil
	}
	return []Agent{*r.agent}, nil
}

func (r *signingRepository) RevokeCredentials(_ context.Context, tenantID, agentID string) error {
	if r.credential != nil && r.credential.TenantID == tenantID && r.credential.AgentID == agentID {
		r.credential.Status = CredentialRevoked
	}
	return nil
}

type countingNonceStore struct{ calls int }

func (s *countingNonceStore) Claim(context.Context, string, string, string, time.Duration) (bool, error) {
	s.calls++
	return true, nil
}

func TestSigningCredentialIsEncryptedAndRequiredForNewCredential(t *testing.T) {
	repo := &signingRepository{}
	protector, err := NewAESGCMProtector([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	nonceStore := &countingNonceStore{}
	svc := NewServiceWithAuditAndNonceStoreAndSigning(repo, nil, nonceStore, time.Minute, 2*time.Minute, protector, false)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops"})
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	if created.RawSigningSecret == "" || len(repo.credential.SigningSecretCiphertext) == 0 {
		t.Fatal("new credential must return a signing secret once and persist ciphertext")
	}
	if strings.Contains(string(repo.credential.SigningSecretCiphertext), created.RawSigningSecret) {
		t.Fatal("signing secret must not be persisted as plaintext")
	}
	secret, err := decodeSigningSecret(created.RawSigningSecret)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"event_id":"evt_1"}`)
	timestamp := time.Now().Unix()
	metadata := AuthenticationMetadata{Timestamp: timestamp, Nonce: "nonce-1", Signature: BuildAgentSignature(secret, "POST", "/api/v1/ingest/events", body, timestamp, "nonce-1"), Method: "POST", Path: "/api/v1/ingest/events", BodyHash: HashRequestBody(body)}
	if _, err := svc.AuthenticateIngestRequest(context.Background(), created.RawAPIKey, metadata); err != nil {
		t.Fatalf("valid signed request error = %v", err)
	}
	if nonceStore.calls != 1 {
		t.Fatalf("nonce claims = %d, want 1", nonceStore.calls)
	}
	metadata.Signature = "v1=bad"
	if _, err := svc.AuthenticateIngestRequest(context.Background(), created.RawAPIKey, metadata); !errors.Is(err, ErrInvalidAgentSignature) {
		t.Fatalf("invalid signature error = %v, want ErrInvalidAgentSignature", err)
	}
	if nonceStore.calls != 1 {
		t.Fatal("invalid signature must not claim a nonce")
	}
}

func TestLegacyCredentialCompatibilityAndRequiredMode(t *testing.T) {
	repo := &signingRepository{}
	svc := NewService(repo)
	created, err := svc.CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Legacy"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "legacy-nonce"}
	svc = NewServiceWithAuditAndNonceStoreAndSigning(repo, nil, &countingNonceStore{}, time.Minute, time.Minute, nil, false)
	if _, err := svc.AuthenticateIngestRequest(context.Background(), created.RawAPIKey, metadata); err != nil {
		t.Fatalf("legacy compatibility error = %v", err)
	}
	svc = NewServiceWithAuditAndNonceStoreAndSigning(repo, nil, &countingNonceStore{}, time.Minute, time.Minute, nil, true)
	if _, err := svc.AuthenticateIngestRequest(context.Background(), created.RawAPIKey, metadata); !errors.Is(err, ErrSignatureRequired) {
		t.Fatalf("required mode error = %v, want ErrSignatureRequired", err)
	}
}
