package agentops

import (
	"encoding/json"
	"time"
)

type Event struct {
	EventID      string          `json:"event_id"`
	TraceID      string          `json:"trace_id"`
	SpanID       string          `json:"span_id"`
	ParentSpanID string          `json:"parent_span_id,omitempty"`
	EventType    string          `json:"event_type"`
	Sequence     int             `json:"sequence,omitempty"`
	OccurredAt   time.Time       `json:"occurred_at"`
	Payload      json.RawMessage `json:"payload"`
}

type IngestResult struct {
	Duplicate bool
}
