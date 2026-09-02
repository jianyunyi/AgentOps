package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platformmetrics "agentscope/internal/platform/metrics"
)

func TestServeMetricsStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	metrics := platformmetrics.NewWorker()
	result := make(chan error, 1)
	go func() { result <- ServeMetrics(ctx, "127.0.0.1:0", metrics) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("ServeMetrics() error = %v, want context canceled", err)
	}
}

func TestMetricsHandlerExposesOnlyReadOnlyHealthAndMetricsRoutes(t *testing.T) {
	handler := MetricsHandler(platformmetrics.NewWorker())

	for _, path := range []string{"/health/live", "/metrics"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, response.Code)
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/metrics", strings.NewReader("payload=secret"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /metrics status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
