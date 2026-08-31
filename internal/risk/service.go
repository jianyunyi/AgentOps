package risk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type StructuredAnalyzer interface {
	Analyze(ctx context.Context, input string) (LLMResult, error)
}

type Service struct {
	repo Repository
	llm  StructuredAnalyzer
}

func NewService(repo Repository, llm StructuredAnalyzer) *Service {
	return &Service{repo: repo, llm: llm}
}

func (s *Service) AnalyzeAndPersist(ctx context.Context, tenantID, traceID, spanID, input string) (Result, error) {
	result := Analyze(input)
	if s.llm != nil {
		llmResult, err := s.llm.Analyze(ctx, result.Redacted)
		if err == nil {
			result = mergeLLMResult(result, llmResult)
		}
	}
	for _, finding := range result.Findings {
		detector := DetectorRules
		if finding.Code == "llm_assessment" {
			detector = DetectorLLM
		}
		event := &RiskEvent{ID: newID("risk_"), TenantID: tenantID, TraceID: traceID, SpanID: spanID, RuleCode: finding.Code, RiskType: finding.Code, RiskLevel: finding.Level, Detector: detector, Reason: finding.Reason, EvidenceRedacted: result.Redacted, Status: RiskOpen, CreatedAt: time.Now().UTC()}
		if err := s.repo.Create(ctx, event); err != nil && !IsDuplicate(err) {
			return Result{}, fmt.Errorf("persist risk event %s: %w", finding.Code, err)
		}
	}
	return result, nil
}

func mergeLLMResult(base Result, llm LLMResult) Result {
	if severity(llm.Level) > severity(base.Level) {
		base.Level = llm.Level
	}
	if llm.Reason != "" {
		base.Findings = append(base.Findings, Finding{Code: "llm_assessment", Level: llm.Level, Reason: llm.Reason})
	}
	return base
}

func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}
