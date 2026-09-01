package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingNonceStore struct{ err error }

func (f failingNonceStore) Claim(context.Context, string, string, string, time.Duration) (bool, error) {
	return false, f.err
}

func newReplayTestService(t *testing.T, store NonceStore) (*Service, string) {
	t.Helper()
	repo := &fakeRepository{}
	created, err := NewService(repo).CreateAgent(context.Background(), CreateAgentInput{TenantID: "tenant_001", Name: "Ops"})
	if err != nil {
		t.Fatal(err)
	}
	return NewServiceWithNonceStore(repo, store, 5*time.Minute, 10*time.Minute), created.RawAPIKey
}

func TestMemoryNonceStoreClaimsOnlyOnce(t *testing.T) {
	store := NewMemoryNonceStore()
	claimed, err := store.Claim(context.Background(), "tenant_001", "agent_001", "nonce-1", time.Minute)
	if err != nil || !claimed {
		t.Fatalf("first Claim() = %v, %v; want true, nil", claimed, err)
	}
	claimed, err = store.Claim(context.Background(), "tenant_001", "agent_001", "nonce-1", time.Minute)
	if err != nil || claimed {
		t.Fatalf("duplicate Claim() = %v, %v; want false, nil", claimed, err)
	}
}

func TestAuthenticateIngestRequestRejectsDuplicateNonce(t *testing.T) {
	service, rawKey := newReplayTestService(t, NewMemoryNonceStore())
	metadata := AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "nonce-1"}
	if _, err := service.AuthenticateIngestRequest(context.Background(), rawKey, metadata); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateIngestRequest(context.Background(), rawKey, metadata); !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("duplicate request error = %v, want ErrReplayDetected", err)
	}
}

func TestAuthenticateIngestRequestRejectsStaleTimestamp(t *testing.T) {
	service, rawKey := newReplayTestService(t, NewMemoryNonceStore())
	_, err := service.AuthenticateIngestRequest(context.Background(), rawKey, AuthenticationMetadata{Timestamp: time.Now().Add(-6 * time.Minute).Unix(), Nonce: "nonce-1"})
	if !errors.Is(err, ErrInvalidAgentRequest) {
		t.Fatalf("stale timestamp error = %v, want ErrInvalidAgentRequest", err)
	}
}

func TestAuthenticateIngestRequestRejectsMalformedNonce(t *testing.T) {
	service, rawKey := newReplayTestService(t, NewMemoryNonceStore())
	_, err := service.AuthenticateIngestRequest(context.Background(), rawKey, AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "bad\nnonce"})
	if !errors.Is(err, ErrInvalidAgentRequest) {
		t.Fatalf("malformed nonce error = %v, want ErrInvalidAgentRequest", err)
	}
}

func TestAuthenticateIngestRequestFailsClosedWithoutNonceStore(t *testing.T) {
	service, rawKey := newReplayTestService(t, nil)
	_, err := service.AuthenticateIngestRequest(context.Background(), rawKey, AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "nonce-1"})
	if !errors.Is(err, ErrNonceStoreUnavailable) {
		t.Fatalf("missing store error = %v, want ErrNonceStoreUnavailable", err)
	}
}

func TestAuthenticateIngestRequestFailsClosedOnNonceStoreError(t *testing.T) {
	service, rawKey := newReplayTestService(t, failingNonceStore{err: errors.New("redis unavailable")})
	_, err := service.AuthenticateIngestRequest(context.Background(), rawKey, AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "nonce-1"})
	if !errors.Is(err, ErrNonceStoreUnavailable) {
		t.Fatalf("store error = %v, want ErrNonceStoreUnavailable", err)
	}
}
