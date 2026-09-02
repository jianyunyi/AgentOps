package agentops

import (
	"context"
	"errors"
	"strconv"
	"time"
)

func shouldRetry(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return isRetryableStatus(apiErr.StatusCode)
	}
	return isRetryableTransportError(err)
}

func retryDelay(policy RetryPolicy, failureAttempt int, err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return minDuration(apiErr.RetryAfter, policy.MaxDelay)
	}
	delay := policy.BaseDelay
	for index := 1; index < failureAttempt; index++ {
		if delay >= policy.MaxDelay || delay > policy.MaxDelay/3 {
			return policy.MaxDelay
		}
		delay *= 3
	}
	return minDuration(delay, policy.MaxDelay)
}

func minDuration(value, maximum time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	if value > maximum {
		return maximum
	}
	return value
}

func waitContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
