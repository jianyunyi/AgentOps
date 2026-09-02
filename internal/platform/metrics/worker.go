package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// WorkerMetrics contains only low-cardinality process-level signals.
type WorkerMetrics struct {
	recovered             atomic.Uint64
	acked                 atomic.Uint64
	retried               atomic.Uint64
	deadLettered          atomic.Uint64
	processingErrors      atomic.Uint64
	outboxPublished       atomic.Uint64
	outboxPublishErrors   atomic.Uint64
	redisClaimErrors      atomic.Uint64
	redisReadErrors       atomic.Uint64
	redisAckErrors        atomic.Uint64
	redisRequeueErrors    atomic.Uint64
	redisDeadLetterErrors atomic.Uint64
	pendingAgeMillis      atomic.Int64
}

func NewWorker() *WorkerMetrics { return &WorkerMetrics{} }

func (m *WorkerMetrics) ObserveRecovered(count int) {
	if count > 0 {
		m.recovered.Add(uint64(count))
	}
}

func (m *WorkerMetrics) ObserveAck()                { m.acked.Add(1) }
func (m *WorkerMetrics) ObserveRetried()            { m.retried.Add(1) }
func (m *WorkerMetrics) ObserveDeadLettered()       { m.deadLettered.Add(1) }
func (m *WorkerMetrics) ObserveProcessingError()    { m.processingErrors.Add(1) }
func (m *WorkerMetrics) ObserveOutboxPublished()    { m.outboxPublished.Add(1) }
func (m *WorkerMetrics) ObserveOutboxPublishError() { m.outboxPublishErrors.Add(1) }

func (m *WorkerMetrics) ObserveRedisError(operation string) {
	switch operation {
	case "claim":
		m.redisClaimErrors.Add(1)
	case "read":
		m.redisReadErrors.Add(1)
	case "ack":
		m.redisAckErrors.Add(1)
	case "requeue":
		m.redisRequeueErrors.Add(1)
	case "dead_letter":
		m.redisDeadLetterErrors.Add(1)
	}
}

func (m *WorkerMetrics) ObserveOutboxPendingAge(age time.Duration) {
	if age <= 0 {
		m.pendingAgeMillis.Store(0)
		return
	}
	m.pendingAgeMillis.Store(age.Milliseconds())
}

func (m *WorkerMetrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		age := float64(m.pendingAgeMillis.Load()) / 1000
		_, _ = fmt.Fprintf(w, "# TYPE agentscope_worker_messages_recovered_total counter\nagentscope_worker_messages_recovered_total %d\n# TYPE agentscope_worker_messages_acked_total counter\nagentscope_worker_messages_acked_total %d\n# TYPE agentscope_worker_messages_retried_total counter\nagentscope_worker_messages_retried_total %d\n# TYPE agentscope_worker_messages_dead_lettered_total counter\nagentscope_worker_messages_dead_lettered_total %d\n# TYPE agentscope_worker_redis_errors_total counter\nagentscope_worker_redis_errors_total{operation=\"claim\"} %d\nagentscope_worker_redis_errors_total{operation=\"read\"} %d\nagentscope_worker_redis_errors_total{operation=\"ack\"} %d\nagentscope_worker_redis_errors_total{operation=\"requeue\"} %d\nagentscope_worker_redis_errors_total{operation=\"dead_letter\"} %d\n# TYPE agentscope_worker_processing_errors_total counter\nagentscope_worker_processing_errors_total %d\n# TYPE agentscope_worker_outbox_published_total counter\nagentscope_worker_outbox_published_total %d\n# TYPE agentscope_worker_outbox_publish_errors_total counter\nagentscope_worker_outbox_publish_errors_total %d\n# TYPE agentscope_worker_outbox_pending_age_seconds gauge\nagentscope_worker_outbox_pending_age_seconds %s\n# TYPE agentscope_worker_info gauge\nagentscope_worker_info{version=\"v1\"} 1\n", m.recovered.Load(), m.acked.Load(), m.retried.Load(), m.deadLettered.Load(), m.redisClaimErrors.Load(), m.redisReadErrors.Load(), m.redisAckErrors.Load(), m.redisRequeueErrors.Load(), m.redisDeadLetterErrors.Load(), m.processingErrors.Load(), m.outboxPublished.Load(), m.outboxPublishErrors.Load(), strconv.FormatFloat(age, 'f', -1, 64))
	}
}
