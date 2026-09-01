package policy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalid = errors.New("invalid policy")

type CreateInput struct {
	TenantID      string
	CreatedBy     string
	Name          string
	RulesEnabled  bool
	LLMEnabled    bool
	MaxInputBytes int
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, input CreateInput) (*Policy, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.CreatedBy) == "" || strings.TrimSpace(input.Name) == "" || input.MaxInputBytes < 1024 || input.MaxInputBytes > 1024*1024 {
		return nil, ErrInvalid
	}
	items, err := s.repo.List(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	item := &Policy{ID: "pol_" + hex.EncodeToString(id), TenantID: input.TenantID, CreatedBy: input.CreatedBy, Name: strings.TrimSpace(input.Name), Version: len(items) + 1, Enabled: len(items) == 0, RulesEnabled: input.RulesEnabled, LLMEnabled: input.LLMEnabled, MaxInputBytes: input.MaxInputBytes}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *Service) List(ctx context.Context, tenantID string) ([]Policy, error) {
	return s.repo.List(ctx, tenantID)
}
func (s *Service) Activate(ctx context.Context, tenantID, id string) error {
	items, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == id {
			return s.repo.Activate(ctx, tenantID, id)
		}
	}
	return ErrNotFound
}
func (s *Service) Active(ctx context.Context, tenantID string) (*Policy, error) {
	return s.repo.Active(ctx, tenantID)
}
