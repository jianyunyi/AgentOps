package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	apihttp "agentscope/internal/http"
	"agentscope/internal/platform/database"
	"agentscope/internal/policy"
	"agentscope/internal/risk"
	"agentscope/internal/tenant"
	"agentscope/internal/trace"
	"gorm.io/gorm"
)

func TestMemberAPIInvitationAndOwnerTransfer(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run API integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	tenantID := fmt.Sprintf("t%d", stamp)
	ownerID := fmt.Sprintf("o%d", stamp)
	targetID := fmt.Sprintf("v%d", stamp)
	if err := db.Create(&tenant.Tenant{ID: tenantID, Name: "API Integration", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	owner := &auth.User{ID: ownerID, TenantID: tenantID, Email: fmt.Sprintf("owner-%d@example.com", stamp), Role: auth.RoleOwner, Status: auth.UserActive}
	target := &auth.User{ID: targetID, TenantID: tenantID, Email: fmt.Sprintf("target-%d@example.com", stamp), Role: auth.RoleDeveloper, Status: auth.UserActive}
	if err := db.Create(owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(target).Error; err != nil {
		t.Fatal(err)
	}
	session := &auth.Session{ID: fmt.Sprintf("s%d", stamp), UserID: ownerID, TenantID: tenantID, Role: auth.RoleOwner, ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	auditService := audit.NewService(audit.NewGORMRepository(db))
	authService := auth.NewServiceWithAudit(auth.NewGORMRepository(db), "integration-secret", auditService)
	agentService := agent.NewService(agent.NewGORMRepository(db))
	riskService := risk.NewService(risk.NewGORMRepository(db), nil)
	traceRepo := trace.NewGORMRepository(db)
	traceService := trace.NewService(traceRepo)
	policyService := policy.NewService(policy.NewGORMRepository(db))
	router := apihttp.NewApplicationRouter(authService, agentService, auditService, riskService, traceService, traceRepo, policyService)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "agentscope_session", Value: session.ID})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	response := request(http.MethodGet, "/api/v1/members?page=1&page_size=1&status=active", "")
	if response.Code != http.StatusOK {
		t.Fatalf("list members status = %d, body=%s", response.Code, response.Body.String())
	}
	var page struct {
		Data       []auth.MemberResponse `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Pagination.Total != 2 {
		t.Fatalf("unexpected page: %+v", page)
	}
	inviteEmail := fmt.Sprintf("new-%d@example.com", stamp)
	response = request(http.MethodPost, "/api/v1/members/invitations", fmt.Sprintf(`{"email":%q,"role":"viewer","ttl_hours":1}`, inviteEmail))
	if response.Code != http.StatusCreated {
		t.Fatalf("invite status = %d, body=%s", response.Code, response.Body.String())
	}
	var invite struct {
		Data struct {
			Token string `json:"invite_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &invite); err != nil {
		t.Fatal(err)
	}
	if invite.Data.Token == "" {
		t.Fatal("invite token must be returned once")
	}
	accept := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invitations/accept", strings.NewReader(fmt.Sprintf(`{"token":%q,"password":"correct-password"}`, invite.Data.Token)))
	accept.Header.Set("Content-Type", "application/json")
	acceptResponse := httptest.NewRecorder()
	router.ServeHTTP(acceptResponse, accept)
	if acceptResponse.Code != http.StatusCreated {
		t.Fatalf("accept status = %d, body=%s", acceptResponse.Code, acceptResponse.Body.String())
	}
	response = request(http.MethodPost, "/api/v1/members/"+targetID+"/transfer-owner", "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("transfer status = %d, body=%s", response.Code, response.Body.String())
	}
	var updated auth.User
	if err := db.Where("id = ?", ownerID).First(&updated).Error; err != nil && err != gorm.ErrRecordNotFound {
		t.Fatal(err)
	}
	if updated.Role != auth.RoleAdmin {
		t.Fatalf("old owner role = %s", updated.Role)
	}
	var auditCount int64
	db.Model(&audit.Record{}).Where("tenant_id = ? AND action IN ?", tenantID, []string{"member.invitation_created", "member.invitation_accepted", "tenant.owner_transferred"}).Count(&auditCount)
	if auditCount != 3 {
		t.Fatalf("audit count = %d, want 3", auditCount)
	}
}
