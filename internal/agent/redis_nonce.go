package agent

import (
	"context"
	"errors"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

type nonceRedisClient interface {
	SetNX(context.Context, string, any, time.Duration) *redisv9.BoolCmd
}

type RedisNonceStore struct {
	client nonceRedisClient
}

func NewRedisNonceStore(client nonceRedisClient) *RedisNonceStore {
	return &RedisNonceStore{client: client}
}

func (s *RedisNonceStore) Claim(ctx context.Context, _ string, agentID, nonce string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, errors.New("redis nonce client is nil")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, errors.New("nonce ttl must be positive")
	}
	return s.client.SetNX(ctx, "agentscope:agent:nonce:"+agentID+":"+nonce, "1", ttl).Result()
}
