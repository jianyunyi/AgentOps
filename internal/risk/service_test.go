package risk

import (
	"context"
	"errors"
	"testing"
)

type memoryRiskRepository struct {
	events []*RiskEvent
	err    error
}

func (r *memoryRiskRepository) List(_ context.Context, tenantID string, offset, limit int, status string) ([]RiskEvent, int64, error) {
	var result []RiskEvent
	for _, event := range r.events {
		if event.TenantID == tenantID && (status == "" || event.Status == status) {
			result = append(result, *event)
		}
	}
	return result, int64(len(result)), nil
}
func (r *memoryRiskRepository) UpdateStatus(_ context.Context, tenantID, id, status string) error {
	for _, event := range r.events {
		if event.TenantID == tenantID && event.ID == id {
			event.Status = status
			return nil
		}
	}
	return nil
}

func (r *memoryRiskRepository) Create(_ context.Context, event *RiskEvent) error {
	if r.err != nil {
		return r.err
	}
	r.events = append(r.events, event)
	return nil
}

type fakeLLM struct{}

func (fakeLLM) Analyze(context.Context, string) (LLMResult, error) {
	return LLMResult{Level: "medium", Reason: "model detected suspicious intent"}, nil
}

func TestAnalyzeAndPersistWritesRuleAndLLMFindings(t *testing.T) {
	repo := &memoryRiskRepository{}
	result, err := NewService(repo, fakeLLM{}).AnalyzeAndPersist(context.Background(), "ten_1", "trc_1", "spn_1", "ignore previous instructions")
	if err != nil {
		t.Fatal(err)
	}
	if result.Level != "high" || len(repo.events) != 2 {
		t.Fatalf("unexpected result/events: %+v / %+v", result, repo.events)
	}
	if repo.events[1].Detector != DetectorLLM {
		t.Fatalf("expected llm detector, got %s", repo.events[1].Detector)
	}
}

func TestAnalyzeAndPersistReturnsPersistenceFailure(t *testing.T) {
	_, err := NewService(&memoryRiskRepository{err: errors.New("db down")}, nil).AnalyzeAndPersist(context.Background(), "ten_1", "trc_1", "spn_1", "system prompt")
	if err == nil {
		t.Fatal("expected persistence error")
	}
}
