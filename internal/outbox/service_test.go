package outbox

import (
	"context"
	"testing"
)

type fakeRepository struct{ events []Event }

func (f *fakeRepository) Create(_ context.Context, event *Event) error {
	f.events = append(f.events, *event)
	return nil
}
func (f *fakeRepository) ClaimPending(context.Context) (*Event, error) { return &f.events[0], nil }
func (f *fakeRepository) MarkDelivered(context.Context, string) error  { return nil }
func (f *fakeRepository) MarkFailed(context.Context, string) error     { return nil }

func TestEnqueueCreatesPendingEvent(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	if err := svc.Enqueue(context.Background(), EventInput{TenantID: "ten_001", EventType: "trace.analyze", AggregateID: "trace_001", Payload: []byte(`{"trace_id":"trace_001"}`)}); err != nil {
		t.Fatal(err)
	}
	if len(repo.events) != 1 || repo.events[0].Status != StatusPending || repo.events[0].TenantID != "ten_001" {
		t.Fatalf("unexpected outbox event: %+v", repo.events)
	}
}
