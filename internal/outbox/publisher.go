package outbox

import "context"

type Sender func(context.Context, Event) error
type Publisher struct {
	repo   Repository
	sender Sender
}

func NewPublisher(repo Repository, sender Sender) *Publisher {
	return &Publisher{repo: repo, sender: sender}
}
func (p *Publisher) PublishOne(ctx context.Context) error {
	event, err := p.repo.ClaimPending(ctx)
	if err != nil {
		return err
	}
	if err := p.sender(ctx, *event); err != nil {
		return p.repo.MarkFailed(ctx, event.ID)
	}
	return p.repo.MarkDelivered(ctx, event.ID)
}
