package auth

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginSetsHttpOnlySessionCookie(t *testing.T) {
	hash, _ := HashPassword("correct-password")
	repo := &fakeUserRepository{users: map[string]*User{
		"owner@example.com": {ID: "usr_001", TenantID: "tenant_001", Email: "owner@example.com", PasswordHash: hash, Role: RoleOwner, Status: UserActive},
	}}
	router := gin.New()
	router.POST("/login", NewHandler(NewService(repo, "test-secret")).Login)

	req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"owner@example.com","password":"correct-password"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatalf("login status = %d, body = %s", res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "agentscope_session" || !cookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly session cookie, got %+v", cookies)
	}
}

func (f *fakeUserRepository) _contextCheck(_ context.Context) {}
