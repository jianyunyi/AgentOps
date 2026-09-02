package outbox

import (
	"context"
	"testing"
	"time"
)

type fakePublisherRepo struct {
	event     Event
	delivered bool
	failed    bool
}

func (f *fakePublisherRepo) Create(context.Context, *Event) error         { return nil }
func (f *fakePublisherRepo) ClaimPending(context.Context) (*Event, error) { return &f.event, nil }
func (f *fakePublisherRepo) MarkDelivered(context.Context, string) error {
	f.delivered = true
	return nil
}
func (f *fakePublisherRepo) MarkFailed(context.Context, string) error { f.failed = true; return nil }

func TestPublisherMarksDeliveredAfterSuccessfulSend(t *testing.T) {
	repo := &fakePublisherRepo{event: Event{ID: "out_001", Payload: []byte(`{}`)}}
	sent := false
	publisher := NewPublisher(repo, func(context.Context, Event) error { sent = true; return nil })
	if err := publisher.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !sent || !repo.delivered || repo.failed {
		t.Fatalf("unexpected publish state: sent=%v delivered=%v failed=%v", sent, repo.delivered, repo.failed)
	}
}

func TestPublisherKeepsFailedEventRetryable(t *testing.T) {
	repo := &fakePublisherRepo{event: Event{ID: "out_001", Payload: []byte(`{}`)}}
	publisher := NewPublisher(repo, func(context.Context, Event) error { return context.DeadlineExceeded })
	if err := publisher.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repo.failed || repo.delivered {
		t.Fatalf("failed event state: delivered=%v failed=%v", repo.delivered, repo.failed)
	}
}

func TestPublisherOutcomeDistinguishesDeliveredAndFailed(t *testing.T) {
	repo := &fakePublisherRepo{event: Event{ID: "out_001", CreatedAt: time.Now().UTC(), Payload: []byte(`{}`)}}
	publisher := NewPublisher(repo, func(context.Context, Event) error { return nil })
	outcome, err := publisher.PublishOneOutcome(context.Background())
	if err != nil || outcome.Status != PublishStatusDelivered {
		t.Fatalf("delivered outcome = %+v, err = %v", outcome, err)
	}

	repo = &fakePublisherRepo{event: Event{ID: "out_002", CreatedAt: time.Now().UTC(), Payload: []byte(`{}`)}}
	publisher = NewPublisher(repo, func(context.Context, Event) error { return context.DeadlineExceeded })
	outcome, err = publisher.PublishOneOutcome(context.Background())
	if err != nil || outcome.Status != PublishStatusFailed {
		t.Fatalf("failed outcome = %+v, err = %v", outcome, err)
	}
}
