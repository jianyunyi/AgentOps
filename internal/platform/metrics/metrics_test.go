package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
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
