package audit

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, input RecordInput) error {
	before, err := marshalRedacted(input.Before)
	if err != nil {
		return err
	}
	after, err := marshalRedacted(input.After)
	if err != nil {
		return err
	}
	actorType := input.ActorType
	if actorType == "" {
		actorType = "user"
	}
	return s.repo.Append(ctx, &Record{TenantID: input.TenantID, ActorID: input.ActorID, ActorType: actorType, Action: input.Action, ResourceType: input.ResourceType, ResourceID: input.ResourceID, Before: before, After: after, RequestID: input.RequestID, CreatedAt: time.Now().UTC()})
}

func marshalRedacted(input map[string]any) (json.RawMessage, error) {
	clean := map[string]any{}
	for key, value := range input {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "api_key") || strings.Contains(lower, "authorization") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") {
			continue
		}
		clean[key] = value
	}
	return json.Marshal(clean)
}
