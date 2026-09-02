package worker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	platformmetrics "agentscope/internal/platform/metrics"
)

const DefaultMetricsAddr = ":9091"

func MetricsHandler(workerMetrics *platformmetrics.WorkerMetrics) http.Handler {
	if workerMetrics == nil {
		workerMetrics = platformmetrics.NewWorker()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", getOnly(workerMetrics.Handler()))
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

func getOnly(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

func ServeMetrics(ctx context.Context, addr string, workerMetrics *platformmetrics.WorkerMetrics) error {
	if addr == "" {
		addr = DefaultMetricsAddr
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           MetricsHandler(workerMetrics),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	}
}
