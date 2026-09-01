package trace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"agentscope/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestAuthenticateQueryRequiresAgentReadPermissionForSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("tenant_id", "tenant_001")
	context.Set("user_role", auth.RoleAuditor)

	identity, ok := (&Handler{}).authenticateQuery(context)
	if ok || identity.TenantID != "" {
		t.Fatal("session without agent:read must not authenticate a trace query")
	}
	if context.Writer.Status() != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", context.Writer.Status(), http.StatusForbidden)
	}
}

func TestAuthenticateQueryRejectsMixedSessionAndAgentCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("tenant_id", "tenant_001")
	context.Set("user_role", auth.RoleOwner)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
	context.Request.Header.Set("Authorization", "Bearer agent-key")

	identity, ok := (&Handler{}).authenticateQuery(context)
	if ok || identity.TenantID != "" || identity.AgentID != "" {
		t.Fatal("mixed session and agent credentials must not authenticate a trace query")
	}
	if context.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", context.Writer.Status(), http.StatusBadRequest)
	}
}
