package agent

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrCredentialNotFound = errors.New("agent credential not found")

type Repository interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	CreateCredential(ctx context.Context, credential *AgentCredential) error
	FindCredentialByHash(ctx context.Context, keyHash string) (*AgentCredential, error)
	ListAgents(ctx context.Context, tenantID string) ([]Agent, error)
}

type GORMRepository struct {
	db *gorm.DB
}

func NewGORMRepository(db *gorm.DB) *GORMRepository {
	return &GORMRepository{db: db}
}

func (r *GORMRepository) CreateAgent(ctx context.Context, agent *Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

func (r *GORMRepository) CreateCredential(ctx context.Context, credential *AgentCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *GORMRepository) FindCredentialByHash(ctx context.Context, keyHash string) (*AgentCredential, error) {
	var credential AgentCredential
	if err := r.db.WithContext(ctx).Where("key_hash = ? AND status = ?", keyHash, CredentialActive).First(&credential).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	return &credential, nil
}

func (r *GORMRepository) ListAgents(ctx context.Context, tenantID string) ([]Agent, error) {
	var agents []Agent
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}
