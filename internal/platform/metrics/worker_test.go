package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWorkerMetricsExposeReliabilitySignals(t *testing.T) {
	metrics := NewWorker()
	metrics.ObserveRecovered(2)
	metrics.ObserveAck()
	metrics.ObserveRetried()
	metrics.ObserveDeadLettered()
	metrics.ObserveRedisError("claim")
	metrics.ObserveRedisError("unknown")
	metrics.ObserveProcessingError()
	metrics.ObserveOutboxPublished()
	metrics.ObserveOutboxPublishError()
	metrics.ObserveOutboxPendingAge(1250 * time.Millisecond)

	response := httptest.NewRecorder()
	metrics.Handler()(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	for _, expected := range []string{
		"agentscope_worker_messages_recovered_total 2",
		"agentscope_worker_messages_acked_total 1",
		"agentscope_worker_messages_retried_total 1",
		"agentscope_worker_messages_dead_lettered_total 1",
		"agentscope_worker_redis_errors_total{operation=\"claim\"} 1",
		"agentscope_worker_processing_errors_total 1",
		"agentscope_worker_outbox_published_total 1",
		"agentscope_worker_outbox_publish_errors_total 1",
		"agentscope_worker_outbox_pending_age_seconds 1.25",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metric %q missing from output: %s", expected, body)
		}
	}
	if strings.Contains(body, "unknown") || strings.Contains(body, "tenant") || strings.Contains(body, "payload") {
		t.Fatalf("worker metrics contain an unexpected high-cardinality or sensitive value: %s", body)
	}
}

func TestWorkerMetricsIgnoreNegativePendingAge(t *testing.T) {
	metrics := NewWorker()
	metrics.ObserveOutboxPendingAge(-time.Second)
	response := httptest.NewRecorder()
	metrics.Handler()(response, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(response.Body.String(), "agentscope_worker_outbox_pending_age_seconds 0") {
		t.Fatalf("negative pending age must be clamped: %s", response.Body.String())
	}
}
