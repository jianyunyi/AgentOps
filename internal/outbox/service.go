package outbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Enqueue(ctx context.Context, input EventInput) error {
	if input.TenantID == "" || input.EventType == "" || input.AggregateID == "" || len(input.Payload) == 0 {
		return errors.New("outbox event fields are required")
	}
	id, err := newID()
	if err != nil {
		return err
	}
	return s.repo.Create(ctx, &Event{ID: id, TenantID: input.TenantID, EventType: input.EventType, AggregateID: input.AggregateID, DedupKey: input.DedupKey, Payload: input.Payload, Status: StatusPending, AvailableAt: time.Now().UTC(), CreatedAt: time.Now().UTC()})
}
func newID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "out_" + hex.EncodeToString(buf), nil
}
