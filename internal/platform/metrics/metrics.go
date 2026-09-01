package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requests atomic.Uint64
	errors   atomic.Uint64
}

func New() *Metrics { return &Metrics{} }

func (m *Metrics) Observe(status int) {
	m.requests.Add(1)
	if status >= 500 {
		m.errors.Add(1)
	}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next(); m.Observe(c.Writer.Status()) }
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# TYPE agentscope_http_requests_total counter\nagentscope_http_requests_total %d\n# TYPE agentscope_http_errors_total counter\nagentscope_http_errors_total %d\n", m.requests.Load(), m.errors.Load())
	}
}
