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
	ID             string     `gorm:"primaryKey;size:32" json:"id"`
	TenantID       string     `gorm:"index;size:32;not null" json:"tenantId"`
	AgentID        string     `gorm:"index;size:32;not null" json:"agentId"`
	TraceID        string     `gorm:"uniqueIndex:idx_trace_tenant;size:64;not null" json:"traceId"`
	Status         string     `gorm:"size:16;not null" json:"status"`
	RiskLevel      string     `gorm:"size:16;not null" json:"riskLevel"`
	TotalTokens    int        `gorm:"not null;default:0" json:"totalTokens"`
	EstimatedCost  float64    `gorm:"not null;default:0" json:"estimatedCost"`
	StartedAt      time.Time  `json:"startedAt"`
	EndedAt        *time.Time `json:"endedAt,omitempty"`
	DurationMS     int64      `gorm:"not null;default:0" json:"durationMs"`
	AnalysisStatus string     `gorm:"size:16;not null" json:"analysisStatus"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Span struct {
	ID             string          `gorm:"primaryKey;size:32" json:"id"`
	TenantID       string          `gorm:"index;size:32;not null" json:"tenantId"`
	TraceID        string          `gorm:"index:idx_span_trace_sequence;size:64;not null" json:"traceId"`
	SpanID         string          `gorm:"uniqueIndex:idx_span_tenant;size:64;not null" json:"spanId"`
	ParentSpanID   string          `gorm:"size:64" json:"parentSpanId"`
	SpanType       string          `gorm:"size:32;not null" json:"spanType"`
	Name           string          `gorm:"size:128;not null" json:"name"`
	Status         string          `gorm:"size:16;not null" json:"status"`
	Sequence       int             `gorm:"index:idx_span_trace_sequence;not null" json:"sequence"`
	InputSnapshot  json.RawMessage `gorm:"type:json" json:"inputSnapshot,omitempty"`
	OutputSnapshot json.RawMessage `gorm:"type:json" json:"outputSnapshot,omitempty"`
	StartedAt      time.Time       `json:"startedAt"`
	EndedAt        *time.Time      `json:"endedAt,omitempty"`
	DurationMS     int64           `gorm:"not null;default:0" json:"durationMs"`
	CreatedAt      time.Time       `json:"createdAt"`
}
