package agent

import (
	"context"
	"errors"
	"time"

	"agentscope/internal/audit"
	"gorm.io/gorm"
)

var (
	ErrCredentialNotFound = errors.New("agent credential not found")
	ErrAgentNotFound      = errors.New("agent not found")
)

type Repository interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	CreateCredential(ctx context.Context, credential *AgentCredential) error
	FindCredentialByHash(ctx context.Context, keyHash string) (*AgentCredential, error)
	FindAgent(ctx context.Context, tenantID, agentID string) (*Agent, error)
	ListAgents(ctx context.Context, tenantID string) ([]Agent, error)
	RevokeCredentials(ctx context.Context, tenantID, agentID string) error
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

func (r *GORMRepository) CreateAgentWithCredentialAndAudit(ctx context.Context, agent *Agent, credential *AgentCredential, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(agent).Error; err != nil {
			return err
		}
		if err := tx.Create(credential).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) RotateCredentialWithAudit(ctx context.Context, tenantID, agentID string, credential *AgentCredential, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&AgentCredential{}).Where("tenant_id = ? AND agent_id = ? AND status = ?", tenantID, agentID, CredentialActive).Updates(map[string]any{"status": CredentialRevoked, "revoked_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		if err := tx.Create(credential).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) RevokeCredentialWithAudit(ctx context.Context, tenantID, agentID string, record *audit.Record) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&AgentCredential{}).Where("tenant_id = ? AND agent_id = ? AND status = ?", tenantID, agentID, CredentialActive).Updates(map[string]any{"status": CredentialRevoked, "revoked_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.Create(record).Error
	})
}

func (r *GORMRepository) FindCredentialByHash(ctx context.Context, keyHash string) (*AgentCredential, error) {
	var credential AgentCredential
	if err := r.db.WithContext(ctx).Where("key_hash = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", keyHash, CredentialActive, time.Now().UTC()).First(&credential).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	return &credential, nil
}

func (r *GORMRepository) FindAgent(ctx context.Context, tenantID, agentID string) (*Agent, error) {
	var agent Agent
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, agentID).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

func (r *GORMRepository) TouchCredential(ctx context.Context, id, ip string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&AgentCredential{}).Where("id = ?", id).Updates(map[string]any{"last_used_at": now, "last_used_ip": ip}).Error
}

func (r *GORMRepository) ListAgents(ctx context.Context, tenantID string) ([]Agent, error) {
	var agents []Agent
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

func (r *GORMRepository) RevokeCredentials(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).Model(&AgentCredential{}).Where("tenant_id = ? AND agent_id = ? AND status = ?", tenantID, agentID, CredentialActive).Updates(map[string]any{"status": CredentialRevoked, "revoked_at": time.Now().UTC()}).Error
}
