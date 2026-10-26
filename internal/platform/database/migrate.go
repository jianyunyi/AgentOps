package database

import (
	"database/sql"
	"fmt"

	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	"agentscope/internal/outbox"
	"agentscope/internal/policy"
	"agentscope/internal/risk"
	"agentscope/internal/tenant"
	"agentscope/internal/trace"
	"gorm.io/gorm"
)

type migration struct {
	Version uint64
	Name    string
	Apply   func(*gorm.DB) error
}

type MigrateOptions struct {
	PurgeLegacySessions bool
}

const (
	migrationLockName        = "agentops_schema_migrations"
	migrationLockWaitSeconds = 60
)

var migrations = []migration{
	{Version: 1, Name: "initial_schema", Apply: migrateInitialSchema},
	{Version: 2, Name: "p0_consistency_and_credentials", Apply: migrateP0Schema},
	{Version: 3, Name: "phase3_members_and_sso", Apply: migratePhase3Schema},
	{Version: 4, Name: "phase5_ai_policies", Apply: migratePhase5Schema},
	{Version: 5, Name: "agent_signing_secret", Apply: migrateAgentSigningSecret},
}

// Migrate applies each schema version exactly once. A MySQL named lock serializes
// startup migrations across API replicas while staying pinned to one connection.
func Migrate(db *gorm.DB) error {
	return MigrateWithOptions(db, MigrateOptions{})
}

func MigrateWithOptions(db *gorm.DB, options MigrateOptions) error {
	return db.Connection(func(conn *gorm.DB) (migrationErr error) {
		var acquired sql.NullInt64
		if err := conn.Raw("SELECT GET_LOCK(?, ?)", migrationLockName, migrationLockWaitSeconds).Row().Scan(&acquired); err != nil {
			return fmt.Errorf("acquire schema migration lock: %w", err)
		}
		if !acquired.Valid || acquired.Int64 != 1 {
			return fmt.Errorf("acquire schema migration lock: timed out after %d seconds", migrationLockWaitSeconds)
		}
		defer func() {
			var released sql.NullInt64
			if err := conn.Raw("SELECT RELEASE_LOCK(?)", migrationLockName).Row().Scan(&released); err != nil {
				migrationErr = fmt.Errorf("release schema migration lock: %w (migration result: %v)", err, migrationErr)
				return
			}
			if !released.Valid || released.Int64 != 1 {
				migrationErr = fmt.Errorf("release schema migration lock: lock was not owned (migration result: %v)", migrationErr)
			}
		}()

		if err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY, name VARCHAR(128) NOT NULL, applied_at DATETIME(6) NOT NULL)`).Error; err != nil {
			return fmt.Errorf("create schema_migrations: %w", err)
		}
		for _, item := range migrations {
			var count int64
			if err := conn.Table("schema_migrations").Where("version = ?", item.Version).Count(&count).Error; err != nil {
				return fmt.Errorf("check migration %d: %w", item.Version, err)
			}
			if count > 0 {
				continue
			}
			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := item.Apply(tx); err != nil {
					return fmt.Errorf("apply migration %d (%s): %w", item.Version, item.Name, err)
				}
				return tx.Exec("INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, UTC_TIMESTAMP(6))", item.Version, item.Name).Error
			}); err != nil {
				return err
			}
		}
		if options.PurgeLegacySessions {
			result := conn.Model(&auth.Session{}).Where("LEFT(id, 4) = ?", "ses_").Delete(&auth.Session{})
			if result.Error != nil {
				return fmt.Errorf("purge legacy plaintext sessions: %w", result.Error)
			}
		}
		return nil
	})
}

func migrateInitialSchema(db *gorm.DB) error {
	return db.AutoMigrate(&agent.Agent{}, &agent.AgentCredential{}, &auth.User{}, &auth.Session{}, &tenant.Tenant{}, &audit.Record{}, &outbox.Event{}, &trace.Trace{}, &trace.Span{}, &trace.EventRecord{}, &risk.RiskEvent{})
}

func migrateP0Schema(db *gorm.DB) error {
	return db.AutoMigrate(&agent.AgentCredential{}, &outbox.Event{}, &risk.RiskEvent{})
}

func migratePhase3Schema(db *gorm.DB) error {
	return db.AutoMigrate(&auth.MemberInvitation{})
}

func migratePhase5Schema(db *gorm.DB) error { return db.AutoMigrate(&policy.Policy{}) }

func migrateAgentSigningSecret(db *gorm.DB) error { return db.AutoMigrate(&agent.AgentCredential{}) }

func MigrationVersions() []uint64 {
	versions := make([]uint64, len(migrations))
	for i, item := range migrations {
		versions[i] = item.Version
	}
	return versions
}
