package database

import (
	"fmt"

	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	"agentscope/internal/outbox"
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

var migrations = []migration{
	{Version: 1, Name: "initial_schema", Apply: migrateInitialSchema},
	{Version: 2, Name: "p0_consistency_and_credentials", Apply: migrateP0Schema},
}

// Migrate applies each schema version exactly once. Production deployments can
// invoke the same function as a dedicated migration job before the new binary.
func Migrate(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY, name VARCHAR(128) NOT NULL, applied_at DATETIME(6) NOT NULL)`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	for _, item := range migrations {
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", item.Version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %d: %w", item.Version, err)
		}
		if count > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := item.Apply(tx); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", item.Version, item.Name, err)
			}
			return tx.Exec("INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, UTC_TIMESTAMP(6))", item.Version, item.Name).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func migrateInitialSchema(db *gorm.DB) error {
	return db.AutoMigrate(&agent.Agent{}, &agent.AgentCredential{}, &auth.User{}, &auth.Session{}, &tenant.Tenant{}, &audit.Record{}, &outbox.Event{}, &trace.Trace{}, &trace.Span{}, &trace.EventRecord{}, &risk.RiskEvent{})
}

func migrateP0Schema(db *gorm.DB) error {
	return db.AutoMigrate(&agent.AgentCredential{}, &outbox.Event{}, &risk.RiskEvent{})
}

func MigrationVersions() []uint64 {
	versions := make([]uint64, len(migrations))
	for i, item := range migrations {
		versions[i] = item.Version
	}
	return versions
}
