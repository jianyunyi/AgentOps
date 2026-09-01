package risk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"agentscope/internal/policy"
)

type StructuredAnalyzer interface {
	Analyze(ctx context.Context, input string) (LLMResult, error)
}

type Service struct {
	repo     Repository
	llm      StructuredAnalyzer
	policies policyProvider
}

type policyProvider interface {
	Active(context.Context, string) (*policy.Policy, error)
}

func (s *Service) List(ctx context.Context, tenantID string, offset, limit int, status string) ([]RiskEvent, int64, error) {
	return s.repo.List(ctx, tenantID, offset, limit, status)
}
func (s *Service) Review(ctx context.Context, tenantID, id, status string) error {
	if status != RiskOpen && status != "acknowledged" && status != "dismissed" && status != "resolved" {
		return fmt.Errorf("invalid risk status")
	}
	return s.repo.UpdateStatus(ctx, tenantID, id, status)
}

func NewService(repo Repository, llm StructuredAnalyzer) *Service {
	return &Service{repo: repo, llm: llm}
}

func NewServiceWithPolicy(repo Repository, llm StructuredAnalyzer, policies policyProvider) *Service {
	return &Service{repo: repo, llm: llm, policies: policies}
}

func (s *Service) AnalyzeAndPersist(ctx context.Context, tenantID, traceID, spanID, input string) (Result, error) {
	configured := &policy.Policy{RulesEnabled: true, LLMEnabled: true, MaxInputBytes: 64 * 1024}
	if s.policies != nil {
		if active, err := s.policies.Active(ctx, tenantID); err == nil && active != nil {
			configured = active
		}
	}
	if configured.MaxInputBytes < 1024 {
		configured.MaxInputBytes = 1024
	}
	if len(input) > configured.MaxInputBytes {
		input = input[:configured.MaxInputBytes]
	}
	result := Analyze(input)
	if !configured.RulesEnabled {
		result = Result{Redacted: result.Redacted}
	}
	if s.llm != nil && configured.LLMEnabled {
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
