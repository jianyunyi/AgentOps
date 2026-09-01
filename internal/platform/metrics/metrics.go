package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requests   atomic.Uint64
	errors     atomic.Uint64
	durationMS atomic.Uint64
}

func (m *Metrics) ObserveDuration(status int, duration time.Duration) {
	m.Observe(status)
	m.durationMS.Add(uint64(duration.Milliseconds()))
}

func New() *Metrics { return &Metrics{} }

func (m *Metrics) Observe(status int) {
	m.requests.Add(1)
	if status >= 500 {
		m.errors.Add(1)
	}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		m.ObserveDuration(c.Writer.Status(), time.Since(started))
	}
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# TYPE agentscope_http_requests_total counter\nagentscope_http_requests_total %d\n# TYPE agentscope_http_errors_total counter\nagentscope_http_errors_total %d\n# TYPE agentscope_http_request_duration_ms_sum counter\nagentscope_http_request_duration_ms_sum %d\n", m.requests.Load(), m.errors.Load(), m.durationMS.Load())
	}
}
