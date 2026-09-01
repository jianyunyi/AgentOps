package ratelimit

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	count int64
	ttl   time.Duration
}

func (s *fakeStore) Increment(context.Context, string, time.Duration) (int64, time.Duration, error) {
	s.count++
	return s.count, s.ttl, nil
}

func TestLimiterRejectsRequestsAfterLimit(t *testing.T) {
	store := &fakeStore{ttl: 3 * time.Second}
	limiter := New(store)
	if allowed, _, err := limiter.Allow(context.Background(), "rl:test", 1, time.Minute); err != nil || !allowed {
		t.Fatalf("first request should pass: %v %v", allowed, err)
	}
	if allowed, retryAfter, err := limiter.Allow(context.Background(), "rl:test", 1, time.Minute); err != nil || allowed || retryAfter <= 0 {
		t.Fatalf("second request should be rejected: %v %v %v", allowed, retryAfter, err)
	}
}
