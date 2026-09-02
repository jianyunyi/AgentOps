package agentops

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPIErrorParsesSafeFields(t *testing.T) {
	secret := "ag_live_secret"
	body := `{"error":{"code":"RATE_LIMITED","message":"retry later"}}`
	headers := make(http.Header)
	headers.Set("X-Request-ID", "req_123")
	headers.Set("Retry-After", "2")
	headers.Set("Authorization", secret)
	err := parseAPIError(http.StatusTooManyRequests, headers, []byte(body))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests || apiErr.Code != "RATE_LIMITED" || apiErr.Message != "retry later" || apiErr.RequestID != "req_123" || apiErr.RetryAfter != 2*time.Second {
		t.Fatalf("api error = %+v", apiErr)
	}
	if strings.Contains(apiErr.Error(), secret) || strings.Contains(apiErr.Error(), body) {
		t.Fatalf("sensitive data leaked in error: %q", apiErr.Error())
	}
}

func TestAPIErrorBoundsUntrustedMessage(t *testing.T) {
	message := strings.Repeat("x", 4096)
	err := parseAPIError(http.StatusInternalServerError, http.Header{}, []byte(`{"error":{"message":"`+message+`"}}`))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Message) > maxErrorMessageLength {
		t.Fatalf("unbounded error message: %+v", apiErr)
	}
}

func TestAPIErrorRedactsRequestSecretsAndBody(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusBadRequest, Message: "api_key=ag_live_secret body={\"prompt\":\"private\"}"}
	redactAPIError(apiErr, "ag_live_secret", `{"prompt":"private"}`)
	if strings.Contains(apiErr.Message, "ag_live_secret") || strings.Contains(apiErr.Message, `{"prompt":"private"}`) {
		t.Fatalf("sensitive data leaked after redaction: %q", apiErr.Message)
	}
}

func TestRetryClassification(t *testing.T) {
	for _, status := range []int{http.StatusRequestTimeout, http.StatusTooManyRequests, 500, 502, 503, 504} {
		if !isRetryableStatus(status) {
			t.Fatalf("status %d should be retryable", status)
		}
	}
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict, http.StatusNotFound} {
		if isRetryableStatus(status) {
			t.Fatalf("status %d should not be retryable", status)
		}
	}
	if !isRetryableTransportError(errors.New("transport failure")) {
		t.Fatal("non-context transport errors should be retryable")
	}
	if isRetryableTransportError(contextCanceledError{}) {
		t.Fatal("context cancellation must not be retried")
	}
}

type contextCanceledError struct{}

func (contextCanceledError) Error() string { return "context canceled" }

func (contextCanceledError) Is(target error) bool { return target == context.Canceled }
