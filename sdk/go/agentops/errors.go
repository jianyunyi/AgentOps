package agentops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxErrorMessageLength = 512

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return "agentscope: unknown api error"
	}
	parts := []string{fmt.Sprintf("agentscope: http status %d", e.StatusCode)}
	if e.Code != "" {
		parts = append(parts, "code="+e.Code)
	}
	if e.Message != "" {
		parts = append(parts, "message="+e.Message)
	}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	return strings.Join(parts, " ")
}

func parseAPIError(status int, headers http.Header, body []byte) error {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	message := truncateErrorMessage(envelope.Error.Message)
	if message == "" {
		message = http.StatusText(status)
	}
	return &APIError{StatusCode: status, Code: truncateErrorMessage(envelope.Error.Code), Message: message, RequestID: sanitizeRequestID(headers.Get("X-Request-ID")), RetryAfter: retryAfter(headers.Get("Retry-After"), time.Now())}
}

func redactAPIError(err error, forbidden ...string) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	for _, value := range forbidden {
		if value != "" {
			apiErr.Message = strings.ReplaceAll(apiErr.Message, value, "[REDACTED]")
		}
	}
	return apiErr
}

func truncateErrorMessage(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
	runes := []rune(value)
	if len(runes) > maxErrorMessageLength {
		return string(runes[:maxErrorMessageLength])
	}
	return value
}

func sanitizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 || strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func retryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		return minRetryDelay(time.Duration(seconds) * time.Second)
	}
	if timestamp, err := http.ParseTime(value); err == nil {
		return minRetryDelay(timestamp.Sub(now))
	}
	return 0
}

func minRetryDelay(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	if value > 10*time.Second {
		return 10 * time.Second
	}
	return value
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableTransportError(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}
