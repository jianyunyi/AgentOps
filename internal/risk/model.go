package risk

import "time"

type RiskEvent struct {
	ID               string    `gorm:"primaryKey;size:32" json:"id"`
	TenantID         string    `gorm:"index;size:32;not null;uniqueIndex:idx_risk_event_once" json:"tenant_id"`
	TraceID          string    `gorm:"index;size:64;not null" json:"trace_id"`
	SpanID           string    `gorm:"index;size:64;not null;uniqueIndex:idx_risk_event_once" json:"span_id"`
	RuleCode         string    `gorm:"size:64;not null;uniqueIndex:idx_risk_event_once" json:"rule_code"`
	RiskType         string    `gorm:"size:32;not null" json:"risk_type"`
	RiskLevel        string    `gorm:"size:16;not null" json:"risk_level"`
	Detector         string    `gorm:"size:32;not null" json:"detector"`
	Reason           string    `gorm:"size:512;not null" json:"reason"`
	EvidenceRedacted string    `gorm:"type:text" json:"evidence_redacted"`
	Status           string    `gorm:"size:16;not null" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

const (
	DetectorRules = "rules"
	DetectorLLM   = "llm"
	RiskOpen      = "open"
)
