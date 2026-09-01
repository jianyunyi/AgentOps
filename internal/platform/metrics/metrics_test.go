package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsHandlerExposesRequestCounters(t *testing.T) {
	metrics := New()
	metrics.Observe(200)
	metrics.Observe(500)
	response := httptest.NewRecorder()
	metrics.Handler()(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	if !strings.Contains(body, "agentscope_http_requests_total 2") || !strings.Contains(body, "agentscope_http_errors_total 1") {
		t.Fatalf("unexpected metrics: %s", body)
	}
}

func TestMetricsHandlerExposesDurationSum(t *testing.T) {
	metrics := New()
	metrics.ObserveDuration(200, 37*time.Millisecond)
	response := httptest.NewRecorder()
	metrics.Handler()(response, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(response.Body.String(), "agentscope_http_request_duration_ms_sum 37") {
		t.Fatalf("duration metric missing: %s", response.Body.String())
	}
}
