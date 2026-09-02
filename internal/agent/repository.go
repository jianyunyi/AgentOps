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

func (r *GORMRepository) GetCredentialMigrationSummary(ctx context.Context, tenantID string) (CredentialMigrationSummary, error) {
	var summary CredentialMigrationSummary
	const query = `
		SELECT
			COUNT(*) AS total_agents,
			SUM(CASE WHEN EXISTS (
				SELECT 1 FROM agent_credentials migrated
				WHERE migrated.agent_id = a.id AND migrated.tenant_id = a.tenant_id AND migrated.status = ?
				AND COALESCE(OCTET_LENGTH(migrated.signing_secret_ciphertext), 0) > 0
			) AND NOT EXISTS (
				SELECT 1 FROM agent_credentials legacy
				WHERE legacy.agent_id = a.id AND legacy.tenant_id = a.tenant_id AND legacy.status = ?
				AND COALESCE(OCTET_LENGTH(legacy.signing_secret_ciphertext), 0) = 0
			) THEN 1 ELSE 0 END) AS migrated_agents,
			SUM(CASE WHEN EXISTS (
				SELECT 1 FROM agent_credentials legacy
				WHERE legacy.agent_id = a.id AND legacy.tenant_id = a.tenant_id AND legacy.status = ?
				AND COALESCE(OCTET_LENGTH(legacy.signing_secret_ciphertext), 0) = 0
			) THEN 1 ELSE 0 END) AS legacy_agents
		FROM agents a
		WHERE a.tenant_id = ? AND a.status = ? AND EXISTS (
			SELECT 1 FROM agent_credentials active
			WHERE active.agent_id = a.id AND active.tenant_id = a.tenant_id AND active.status = ?
		)`
	if err := r.db.WithContext(ctx).Raw(query, CredentialActive, CredentialActive, CredentialActive, tenantID, AgentStatusActive, CredentialActive).Scan(&summary).Error; err != nil {
		return CredentialMigrationSummary{}, err
	}
	return summary, nil
}

func (r *GORMRepository) ListLegacyAgents(ctx context.Context, tenantID, query string, page, pageSize int) ([]LegacyAgent, int64, error) {
	var total int64
	base := r.db.WithContext(ctx).Table("agents AS a").
		Joins("JOIN agent_credentials AS c ON c.agent_id = a.id AND c.tenant_id = a.tenant_id").
		Where("a.tenant_id = ? AND a.status = ? AND c.status = ? AND COALESCE(OCTET_LENGTH(c.signing_secret_ciphertext), 0) = 0", tenantID, AgentStatusActive, CredentialActive)
	if query != "" {
		like := "%" + query + "%"
		base = base.Where("(a.id LIKE ? OR a.name LIKE ?)", like, like)
	}
	countQuery := base.Session(&gorm.Session{}).Select("COUNT(DISTINCT a.id)").Count(&total)
	if countQuery.Error != nil {
		return nil, 0, countQuery.Error
	}
	type legacyAgentRow struct {
		ID          string     `gorm:"column:id"`
		Name        string     `gorm:"column:name"`
		Environment string     `gorm:"column:environment"`
		Status      string     `gorm:"column:status"`
		LastUsedAt  *time.Time `gorm:"column:last_used_at"`
	}
	var rows []legacyAgentRow
	offset := (page - 1) * pageSize
	listQuery := base.Select("a.id, a.name, a.environment, a.status, MAX(c.last_used_at) AS last_used_at").
		Group("a.id, a.name, a.environment, a.status, a.created_at").
		Order("a.created_at DESC").Offset(offset).Limit(pageSize).Scan(&rows)
	if listQuery.Error != nil {
		return nil, 0, listQuery.Error
	}
	agents := make([]LegacyAgent, 0, len(rows))
	for _, row := range rows {
		agents = append(agents, LegacyAgent{ID: row.ID, Name: row.Name, Environment: row.Environment, Status: row.Status, LastUsedAt: row.LastUsedAt})
	}
	return agents, total, nil
}

func (r *GORMRepository) CountLegacyCredentials(ctx context.Context) (int64, error) {
	var count int64
	const query = `
		SELECT COUNT(DISTINCT a.id)
		FROM agents a
		JOIN agent_credentials c ON c.agent_id = a.id AND c.tenant_id = a.tenant_id
		WHERE a.status = ? AND c.status = ? AND COALESCE(OCTET_LENGTH(c.signing_secret_ciphertext), 0) = 0`
	if err := r.db.WithContext(ctx).Raw(query, AgentStatusActive, CredentialActive).Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
