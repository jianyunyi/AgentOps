package policy

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("policy not found")

type Repository interface {
	Create(context.Context, *Policy) error
	List(context.Context, string) ([]Policy, error)
	Activate(context.Context, string, string) error
	Active(context.Context, string) (*Policy, error)
}

type GORMRepository struct{ db *gorm.DB }

func NewGORMRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) Create(ctx context.Context, policy *Policy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}
func (r *GORMRepository) List(ctx context.Context, tenantID string) ([]Policy, error) {
	var items []Policy
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("version desc").Find(&items).Error
	return items, err
}
func (r *GORMRepository) Active(ctx context.Context, tenantID string) (*Policy, error) {
	var item Policy
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND enabled = ?", tenantID, true).Order("version desc").First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}
func (r *GORMRepository) Activate(ctx context.Context, tenantID, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target Policy
		if err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Model(&Policy{}).Where("tenant_id = ?", tenantID).Update("enabled", false).Error; err != nil {
			return err
		}
		return tx.Model(&Policy{}).Where("id = ? AND tenant_id = ?", id, tenantID).Update("enabled", true).Error
	})
}
