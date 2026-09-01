package policy

import (
	"context"
	"testing"
)

type fakeRepository struct{ policies []Policy }

func (f *fakeRepository) Create(_ context.Context, p *Policy) error {
	f.policies = append(f.policies, *p)
	return nil
}
func (f *fakeRepository) List(_ context.Context, tenantID string) ([]Policy, error) {
	var out []Policy
	for _, p := range f.policies {
		if p.TenantID == tenantID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *fakeRepository) Activate(_ context.Context, tenantID, id string) error {
	for i := range f.policies {
		if f.policies[i].TenantID == tenantID {
			f.policies[i].Enabled = f.policies[i].ID == id
		}
	}
	return nil
}
func (f *fakeRepository) Active(_ context.Context, tenantID string) (*Policy, error) {
	for i := range f.policies {
		if f.policies[i].TenantID == tenantID && f.policies[i].Enabled {
			p := f.policies[i]
			return &p, nil
		}
	}
	return nil, ErrNotFound
}

func TestCreateRejectsUnsafeLimits(t *testing.T) {
	svc := NewService(&fakeRepository{})
	if _, err := svc.Create(context.Background(), CreateInput{TenantID: "ten_1", Name: "default", MaxInputBytes: 0}); err == nil {
		t.Fatal("zero max input must be rejected")
	}
}

func TestActivateIsTenantScoped(t *testing.T) {
	repo := &fakeRepository{policies: []Policy{{ID: "pol_1", TenantID: "ten_1", Enabled: true}, {ID: "pol_2", TenantID: "ten_2", Enabled: true}}}
	if err := NewService(repo).Activate(context.Background(), "ten_1", "pol_2"); err == nil {
		t.Fatal("cross-tenant activation must fail")
	}
}
