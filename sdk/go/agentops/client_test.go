package agentops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

func testClient(t *testing.T, serverURL string, retry RetryPolicy) *Client {
	t.Helper()
	secret := base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	client, err := NewClient(Config{BaseURL: serverURL, APIKey: "ag_live_test", SigningSecret: secret, Retry: retry})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testEvent() Event {
	return Event{EventID: "evt_1", TraceID: "trace_1", SpanID: "span_1", EventType: "llm_call", OccurredAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Payload: json.RawMessage(`{"prompt":"hello"}`)}
}

func TestIngestSignsExactBodyAndParsesDuplicate(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/ingest/events" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &received); err != nil {
			t.Fatal(err)
		}
		timestamp, err := strconv.ParseInt(r.Header.Get("X-Agent-Timestamp"), 10, 64)
		if err != nil {
			t.Fatalf("timestamp = %q", r.Header.Get("X-Agent-Timestamp"))
		}
		secret := []byte("01234567890123456789012345678901")
		expected := signRequest(secret, r.Method, r.URL.Path, body, timestamp, r.Header.Get("X-Agent-Nonce"))
		if r.Header.Get("Authorization") != "Bearer ag_live_test" || r.Header.Get("X-Agent-Signature") != expected || r.Header.Get("X-Agent-Nonce") == "" || r.Header.Get("X-Request-ID") == "" {
			t.Fatalf("auth headers = %+v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"duplicate":true}}`))
	}))
	defer server.Close()

	result, err := testClient(t, server.URL, RetryPolicy{MaxAttempts: 1}).Ingest(context.Background(), testEvent())
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if !result.Duplicate || received.EventID != "evt_1" {
		t.Fatalf("result = %+v received = %+v", result, received)
	}
}

func TestIngestRetriesWithFreshNonceAndStableBody(t *testing.T) {
	var mu sync.Mutex
	var bodies [][]byte
	var nonces []string
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, body)
		nonces = append(nonces, r.Header.Get("X-Agent-Nonce"))
		attempt++
		current := attempt
		mu.Unlock()
		if current == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"duplicate":false}}`))
	}))
	defer server.Close()

	result, err := testClient(t, server.URL, RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}).Ingest(context.Background(), testEvent())
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Duplicate || len(bodies) != 2 || string(bodies[0]) != string(bodies[1]) || nonces[0] == nonces[1] {
		t.Fatalf("bodies = %q/%q nonces = %q/%q result = %+v", bodies[0], bodies[1], nonces[0], nonces[1], result)
	}
}

func TestIngestDoesNotRetryNonRetryableStatus(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusConflict, http.StatusBadRequest} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"code":"FAILED","message":"no retry"}}`))
			}))
			defer server.Close()
			_, err := testClient(t, server.URL, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}).Ingest(context.Background(), testEvent())
			if err == nil || calls != 1 {
				t.Fatalf("status %d err = %v calls = %d", status, err, calls)
			}
		})
	}
}

func TestIngestBackoffHonorsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := testClient(t, server.URL, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: time.Second}).Ingest(ctx, testEvent())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ingest() error = %v, want context deadline", err)
	}
}
