package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

type fakeNonceRedisClient struct {
	key        string
	value      any
	expiration time.Duration
	claimed    bool
	err        error
}

func (f *fakeNonceRedisClient) SetNX(_ context.Context, key string, value any, expiration time.Duration) *redisv9.BoolCmd {
	f.key = key
	f.value = value
	f.expiration = expiration
	return redisv9.NewBoolResult(f.claimed, f.err)
}

func TestRedisNonceStoreClaimsWithNamespacedKeyAndTTL(t *testing.T) {
	client := &fakeNonceRedisClient{claimed: true}
	store := NewRedisNonceStore(client)
	claimed, err := store.Claim(context.Background(), "tenant_001", "agent_001", "nonce-1", 10*time.Minute)
	if err != nil || !claimed {
		t.Fatalf("Claim() = %v, %v; want true, nil", claimed, err)
	}
	if client.key != "agentscope:agent:nonce:tenant_001:agent_001:nonce-1" || client.value != "1" || client.expiration != 10*time.Minute {
		t.Fatalf("SETNX args = key:%q value:%v ttl:%v", client.key, client.value, client.expiration)
	}
}

func TestRedisNonceStoreReturnsRedisError(t *testing.T) {
	client := &fakeNonceRedisClient{err: errors.New("redis unavailable")}
	store := NewRedisNonceStore(client)
	if _, err := store.Claim(context.Background(), "tenant_001", "agent_001", "nonce-1", time.Minute); err == nil {
		t.Fatal("Claim() must return Redis errors")
	}
}
