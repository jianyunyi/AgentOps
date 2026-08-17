package worker

import (
	"context"
	"errors"
	"testing"
)

func TestAnalysisProcessorAcknowledgesSuccessfulMessage(t *testing.T) {
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return nil
	})

	decision := processor.Process(context.Background(), AnalysisMessage{EventID: "evt_001"}, 1)
	if decision != DecisionAck {
		t.Fatalf("successful message decision = %s, want %s", decision, DecisionAck)
	}
}

func TestAnalysisProcessorRetriesTransientFailure(t *testing.T) {
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return errors.New("temporary dependency failure")
	})

	decision := processor.Process(context.Background(), AnalysisMessage{EventID: "evt_001"}, 1)
	if decision != DecisionRetry {
		t.Fatalf("first failure decision = %s, want %s", decision, DecisionRetry)
	}
}

func TestAnalysisProcessorDeadLettersAfterMaximumAttempts(t *testing.T) {
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		return errors.New("permanent failure")
	})

	decision := processor.Process(context.Background(), AnalysisMessage{EventID: "evt_001"}, 3)
	if decision != DecisionDeadLetter {
		t.Fatalf("maximum-attempt decision = %s, want %s", decision, DecisionDeadLetter)
	}
}

func TestAnalysisProcessorDoesNotReprocessCompletedEvent(t *testing.T) {
	called := 0
	processor := NewAnalysisProcessor(3, func(context.Context, AnalysisMessage) error {
		called++
		return nil
	})
	message := AnalysisMessage{EventID: "evt_001"}

	if got := processor.Process(context.Background(), message, 1); got != DecisionAck {
		t.Fatalf("first decision = %s, want %s", got, DecisionAck)
	}
	if got := processor.Process(context.Background(), message, 1); got != DecisionAck {
		t.Fatalf("duplicate decision = %s, want %s", got, DecisionAck)
	}
	if called != 1 {
		t.Fatalf("analyzer called %d times, want 1", called)
	}
}
