package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Sender func(context.Context, Event) error

type PublishStatus string

const (
	PublishStatusNoEvent   PublishStatus = "no_event"
	PublishStatusDelivered PublishStatus = "delivered"
	PublishStatusFailed    PublishStatus = "failed"
)

type PublishOutcome struct {
	Status     PublishStatus
	PendingAge time.Duration
}

type Publisher struct {
	repo   Repository
	sender Sender
}

func NewPublisher(repo Repository, sender Sender) *Publisher {
	return &Publisher{repo: repo, sender: sender}
}
func (p *Publisher) PublishOne(ctx context.Context) error {
	_, err := p.PublishOneOutcome(ctx)
	return err
}

func (p *Publisher) PublishOneOutcome(ctx context.Context) (PublishOutcome, error) {
	event, err := p.repo.ClaimPending(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PublishOutcome{Status: PublishStatusNoEvent}, nil
		}
		return PublishOutcome{Status: PublishStatusFailed}, err
	}
	if event == nil {
		return PublishOutcome{Status: PublishStatusFailed}, fmt.Errorf("outbox repository returned a nil event")
	}
	outcome := PublishOutcome{Status: PublishStatusFailed}
	if !event.CreatedAt.IsZero() {
		outcome.PendingAge = time.Since(event.CreatedAt)
		if outcome.PendingAge < 0 {
			outcome.PendingAge = 0
		}
	}
	if err := p.sender(ctx, *event); err != nil {
		return outcome, p.repo.MarkFailed(ctx, event.ID)
	}
	if err := p.repo.MarkDelivered(ctx, event.ID); err != nil {
		return outcome, err
	}
	outcome.Status = PublishStatusDelivered
	return outcome, nil
}
