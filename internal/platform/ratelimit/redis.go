package ratelimit

import (
	"context"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

type Store interface {
	Increment(context.Context, string, time.Duration) (int64, time.Duration, error)
}

type Limiter struct{ store Store }

func New(store Store) *Limiter { return &Limiter{store: store} }

func (l *Limiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, time.Duration, error) {
	if limit < 1 || window <= 0 {
		return false, 0, fmt.Errorf("invalid rate limit configuration")
	}
	count, ttl, err := l.store.Increment(ctx, key, window)
	if err != nil {
		return false, 0, err
	}
	if count > limit {
		if ttl < time.Second {
			ttl = time.Second
		}
		return false, ttl, nil
	}
	return true, 0, nil
}

type RedisStore struct{ client redisv9.UniversalClient }

func NewRedisStore(client redisv9.UniversalClient) *RedisStore { return &RedisStore{client: client} }

var incrementScript = redisv9.NewScript(`local count = redis.call('INCR', KEYS[1]); if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end; return {count, redis.call('PTTL', KEYS[1])}`)

func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	values, err := incrementScript.Run(ctx, s.client, []string{key}, window.Milliseconds()).Result()
	if err != nil {
		return 0, 0, err
	}
	items, ok := values.([]any)
	if !ok || len(items) != 2 {
		return 0, 0, fmt.Errorf("unexpected redis rate limit response")
	}
	count, ok := items[0].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("invalid redis rate limit count")
	}
	ttl, ok := items[1].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("invalid redis rate limit ttl")
	}
	return count, time.Duration(ttl) * time.Millisecond, nil
}
