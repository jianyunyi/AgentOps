package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"agentscope/internal/agent"
	"agentscope/internal/platform/database"
)

func TestMySQLAgentCredentialMigrationQueries(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run MySQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.Open(ctx, os.Getenv("MYSQL_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UnixNano()
	tenantID := fmt.Sprintf("mt%d", stamp)
	legacyID := fmt.Sprintf("ml%d", stamp)
	migratedID := fmt.Sprintf("mm%d", stamp)
	revokedID := fmt.Sprintf("mr%d", stamp)
	defer db.Where("tenant_id = ?", tenantID).Delete(&agent.Agent{})
	defer db.Where("tenant_id = ?", tenantID).Delete(&agent.AgentCredential{})
	agents := []agent.Agent{
		{ID: legacyID, TenantID: tenantID, Name: "Legacy", Environment: "production", Status: agent.AgentStatusActive},
		{ID: migratedID, TenantID: tenantID, Name: "Migrated", Environment: "production", Status: agent.AgentStatusActive},
		{ID: revokedID, TenantID: tenantID, Name: "Revoked", Environment: "production", Status: agent.AgentStatusActive},
	}
	if err := db.Create(&agents).Error; err != nil {
		t.Fatal(err)
	}
	credentials := []agent.AgentCredential{
		{ID: fmt.Sprintf("mcl%d", stamp), TenantID: tenantID, AgentID: legacyID, KeyPrefix: "legacy", KeyHash: fmt.Sprintf("legacy_hash_%d", stamp), Status: agent.CredentialActive},
		{ID: fmt.Sprintf("mcm%d", stamp), TenantID: tenantID, AgentID: migratedID, KeyPrefix: "migrated", KeyHash: fmt.Sprintf("migrated_hash_%d", stamp), SigningSecretCiphertext: []byte{1, 2, 3}, Status: agent.CredentialActive},
		{ID: fmt.Sprintf("mcr%d", stamp), TenantID: tenantID, AgentID: revokedID, KeyPrefix: "revoked", KeyHash: fmt.Sprintf("revoked_hash_%d", stamp), Status: agent.CredentialRevoked},
	}
	if err := db.Create(&credentials).Error; err != nil {
		t.Fatal(err)
	}
	repo := agent.NewGORMRepository(db)
	summary, err := repo.GetCredentialMigrationSummary(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalAgents != 2 || summary.MigratedAgents != 1 || summary.LegacyAgents != 1 {
		t.Fatalf("summary = %+v, want total=2 migrated=1 legacy=1", summary)
	}
	agentsPage, total, err := repo.ListLegacyAgents(ctx, tenantID, "Legacy", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(agentsPage) != 1 || agentsPage[0].ID != legacyID {
		t.Fatalf("legacy page = %+v total=%d", agentsPage, total)
	}
	count, err := repo.CountLegacyCredentials(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("global legacy count = %d, want at least 1", count)
	}
}
