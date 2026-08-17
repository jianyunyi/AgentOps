package trace

import (
	"encoding/json"
	"time"
)

const MaxPayloadBytes = 1024 * 1024

const (
	EventTraceStart  = "trace_start"
	EventLLMCall     = "llm_call"
	EventToolCall    = "tool_call"
	EventRiskCheck   = "risk_check"
	EventAgentOutput = "agent_output"
	EventTraceEnd    = "trace_end"
)

const (
	TraceRunning = "running"
	TraceSuccess = "success"
	TraceFailed  = "failed"
	TraceTimeout = "timeout"
)

type Event struct {
	EventID      string          `json:"event_id"`
	TraceID      string          `json:"trace_id"`
	SpanID       string          `json:"span_id"`
	ParentSpanID string          `json:"parent_span_id"`
	EventType    string          `json:"event_type"`
	Sequence     int             `json:"sequence"`
	OccurredAt   time.Time       `json:"occurred_at"`
	Payload      json.RawMessage `json:"payload"`
}

type EventRecord struct {
	ID        uint   `gorm:"primaryKey"`
	TenantID  string `gorm:"uniqueIndex:idx_event_tenant;size:32;not null"`
	EventID   string `gorm:"uniqueIndex:idx_event_tenant;size:64;not null"`
	CreatedAt time.Time
}

type IngestContext struct {
	TenantID string
	AgentID  string
}

type IngestResult struct {
	Duplicate bool
}

type Trace struct {
	ID             string  `gorm:"primaryKey;size:32"`
	TenantID       string  `gorm:"index;size:32;not null"`
	AgentID        string  `gorm:"index;size:32;not null"`
	TraceID        string  `gorm:"uniqueIndex:idx_trace_tenant;size:64;not null"`
	Status         string  `gorm:"size:16;not null"`
	RiskLevel      string  `gorm:"size:16;not null"`
	TotalTokens    int     `gorm:"not null;default:0"`
	EstimatedCost  float64 `gorm:"not null;default:0"`
	StartedAt      time.Time
	EndedAt        *time.Time
	DurationMS     int64  `gorm:"not null;default:0"`
	AnalysisStatus string `gorm:"size:16;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Span struct {
	ID             string          `gorm:"primaryKey;size:32"`
	TenantID       string          `gorm:"index;size:32;not null"`
	TraceID        string          `gorm:"index:idx_span_trace_sequence;size:64;not null"`
	SpanID         string          `gorm:"uniqueIndex:idx_span_tenant;size:64;not null"`
	ParentSpanID   string          `gorm:"size:64"`
	SpanType       string          `gorm:"size:32;not null"`
	Name           string          `gorm:"size:128;not null"`
	Status         string          `gorm:"size:16;not null"`
	Sequence       int             `gorm:"index:idx_span_trace_sequence;not null"`
	InputSnapshot  json.RawMessage `gorm:"type:json"`
	OutputSnapshot json.RawMessage `gorm:"type:json"`
	StartedAt      time.Time
	EndedAt        *time.Time
	DurationMS     int64 `gorm:"not null;default:0"`
	CreatedAt      time.Time
}
