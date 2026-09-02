package worker

import (
	"context"
	"sync"

	"agentscope/internal/risk"
)

type Decision string

const (
	DecisionAck        Decision = "ack"
	DecisionRetry      Decision = "retry"
	DecisionDeadLetter Decision = "dead_letter"
)

type AnalysisMessage struct {
	Version  string `json:"version"`
	TenantID string `json:"tenant_id"`
	EventID  string `json:"event_id"`
	TraceID  string `json:"trace_id"`
	SpanID   string `json:"span_id"`
	Input    string `json:"input"`
}

type Analyzer func(ctx context.Context, message AnalysisMessage) error

type RiskResult struct {
	Level    string
	Findings []risk.Finding
	Redacted string
}

func AnalyzeRiskPayload(payload string) RiskResult {
	result := risk.Analyze(payload)
	return RiskResult{Level: result.Level, Findings: result.Findings, Redacted: result.Redacted}
}

type AnalysisProcessor struct {
	maxAttempts int
	analyzer    Analyzer
	completed   map[string]struct{}
	mu          sync.Mutex
}

func NewAnalysisProcessor(maxAttempts int, analyzer Analyzer) *AnalysisProcessor {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &AnalysisProcessor{
		maxAttempts: maxAttempts,
		analyzer:    analyzer,
		completed:   make(map[string]struct{}),
	}
}

func (p *AnalysisProcessor) Process(ctx context.Context, message AnalysisMessage, attempt int) Decision {
	decision, _ := p.ProcessWithError(ctx, message, attempt)
	return decision
}

func (p *AnalysisProcessor) ProcessWithError(ctx context.Context, message AnalysisMessage, attempt int) (Decision, error) {
	if message.EventID == "" {
		return DecisionDeadLetter, nil
	}
	key := message.TenantID + ":" + message.EventID
	p.mu.Lock()
	if _, ok := p.completed[message.TenantID+":"+message.EventID]; ok {
		p.mu.Unlock()
		return DecisionAck, nil
	}
	p.mu.Unlock()
	if err := p.analyzer(ctx, message); err != nil {
		if attempt >= p.maxAttempts {
			return DecisionDeadLetter, err
		}
		return DecisionRetry, err
	}
	p.mu.Lock()
	p.completed[key] = struct{}{}
	p.mu.Unlock()
	return DecisionAck, nil
}
