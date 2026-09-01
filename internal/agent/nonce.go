package agent

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidAgentRequest   = errors.New("invalid agent request")
	ErrReplayDetected        = errors.New("agent request replay detected")
	ErrNonceStoreUnavailable = errors.New("agent nonce store unavailable")
)

type AuthenticationMetadata struct {
	Timestamp int64
	Nonce     string
}

type NonceStore interface {
	Claim(context.Context, string, string, string, time.Duration) (bool, error)
}

type MemoryNonceStore struct {
	mu     sync.Mutex
	values map[string]time.Time
}

func NewMemoryNonceStore() *MemoryNonceStore {
	return &MemoryNonceStore{values: make(map[string]time.Time)}
}

func (s *MemoryNonceStore) Claim(ctx context.Context, tenantID, agentID, nonce string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, errors.New("nonce ttl must be positive")
	}

	now := time.Now().UTC()
	key := tenantID + ":" + agentID + ":" + nonce
	s.mu.Lock()
	defer s.mu.Unlock()
	if expiresAt, exists := s.values[key]; exists && expiresAt.After(now) {
		return false, nil
	}
	s.values[key] = now.Add(ttl)
	return true, nil
}
