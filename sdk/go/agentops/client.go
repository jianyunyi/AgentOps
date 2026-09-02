package agentops

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ingestPath           = "/api/v1/ingest/events"
	maxResponseBodyBytes = 64 * 1024
)

func (c *Client) Ingest(ctx context.Context, event Event) (IngestResult, error) {
	if err := validateEvent(event); err != nil {
		return IngestResult{}, err
	}
	body, err := json.Marshal(event)
	if err != nil {
		return IngestResult{}, errors.New("marshal agent event")
	}
	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		result, requestErr := c.ingestAttempt(ctx, body)
		if requestErr == nil {
			return result, nil
		}
		if attempt == c.retry.MaxAttempts || ctx.Err() != nil || !shouldRetry(requestErr) {
			return IngestResult{}, requestErr
		}
		if err := waitContext(ctx, retryDelay(c.retry, attempt, requestErr)); err != nil {
			return IngestResult{}, err
		}
	}
	return IngestResult{}, errors.New("agent ingest attempts exhausted")
}

func (c *Client) ingestAttempt(ctx context.Context, body []byte) (IngestResult, error) {
	timestamp := time.Now().Unix()
	nonce, err := newNonce()
	if err != nil {
		return IngestResult{}, errors.New("generate agent request nonce")
	}
	requestID, err := newNonce()
	if err != nil {
		return IngestResult{}, errors.New("generate request id")
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + ingestPath
	endpoint.RawPath = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return IngestResult{}, errors.New("create agent ingest request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("X-Agent-Timestamp", formatUnixTimestamp(timestamp))
	request.Header.Set("X-Agent-Nonce", nonce)
	request.Header.Set("X-Agent-Signature", signRequest(c.signingSecret, http.MethodPost, endpoint.Path, body, timestamp, nonce))
	request.Header.Set("X-Request-ID", "req_"+requestID)
	request.Header.Set("User-Agent", c.userAgent)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return IngestResult{}, err
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return IngestResult{}, errors.New("read agent ingest response")
	}
	if len(responseBody) > maxResponseBodyBytes {
		return IngestResult{}, &APIError{StatusCode: response.StatusCode, Code: "RESPONSE_TOO_LARGE", Message: "agent response is too large", RequestID: sanitizeRequestID(response.Header.Get("X-Request-ID"))}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		return IngestResult{}, redactAPIError(parseAPIError(response.StatusCode, response.Header, responseBody), c.apiKey, string(body), base64.RawStdEncoding.EncodeToString(c.signingSecret), base64.StdEncoding.EncodeToString(c.signingSecret))
	}
	var envelope struct {
		Data *struct {
			Duplicate bool `json:"duplicate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil || envelope.Data == nil {
		return IngestResult{}, &APIError{StatusCode: response.StatusCode, Code: "INVALID_RESPONSE", Message: "agent response is invalid", RequestID: sanitizeRequestID(response.Header.Get("X-Request-ID"))}
	}
	return IngestResult{Duplicate: envelope.Data.Duplicate}, nil
}

func validateEvent(event Event) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SpanID) == "" || strings.TrimSpace(event.EventType) == "" {
		return errors.New("event id, trace id, span id, and event type are required")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("event occurred at is required")
	}
	return nil
}

func formatUnixTimestamp(value int64) string {
	return formatInt64(value)
}
