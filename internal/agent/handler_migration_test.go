package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMigrationStatusHandlerDoesNotExposeSecrets(t *testing.T) {
	repo := &migrationRepository{
		summary: CredentialMigrationSummary{TotalAgents: 2, MigratedAgents: 1, LegacyAgents: 1},
		agents:  []LegacyAgent{{ID: "agt_legacy", Name: "Legacy Agent", Environment: "production", Status: AgentStatusActive}},
		total:   1,
	}
	router := gin.New()
	h := NewHandler(NewService(repo))
	router.GET("/migration-status", func(c *gin.Context) {
		c.Set("tenant_id", "tenant_001")
		h.MigrationStatus(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/migration-status?page=1&page_size=20&q=legacy", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); body == "" || strings.Contains(body, "api_key") || strings.Contains(body, "signing_secret") || strings.Contains(body, "key_hash") || strings.Contains(body, "ciphertext") {
		t.Fatalf("migration response contains secret fields: %s", body)
	}
}

func TestMigrationStatusHandlerMapsUnavailableRepository(t *testing.T) {
	router := gin.New()
	h := NewHandler(NewService(&fakeRepository{}))
	router.GET("/migration-status", func(c *gin.Context) {
		c.Set("tenant_id", "tenant_001")
		h.MigrationStatus(c)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/migration-status", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "MIGRATION_STATUS_UNAVAILABLE") {
		t.Fatalf("unexpected unavailable response: status=%d body=%s", response.Code, response.Body.String())
	}
}

var _ interface {
	GetCredentialMigrationSummary(context.Context, string) (CredentialMigrationSummary, error)
	ListLegacyAgents(context.Context, string, string, int, int) ([]LegacyAgent, int64, error)
} = (*migrationRepository)(nil)
