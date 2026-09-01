package auth

import (
	"context"
	"net/http"
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
	if cookies[0].Secure != true || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie security flags = secure:%v sameSite:%v", cookies[0].Secure, cookies[0].SameSite)
	}
}

func TestLogoutClearsSessionCookie(t *testing.T) {
	handler := NewHandler(NewService(&fakeUserRepository{}, "test-secret"))
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	handler.Logout(context)
	cookies := context.Writer.Header().Values("Set-Cookie")
	if len(cookies) != 1 || !strings.Contains(cookies[0], "Max-Age=0") {
		t.Fatalf("expected expired session cookie, got %v", cookies)
	}
}

func TestRegisterCreatesSessionCookie(t *testing.T) {
	repo := &fakeUserRepository{users: map[string]*User{}}
	router := gin.New()
	router.POST("/register", NewHandler(NewService(repo, "test-secret")).Register)
	req := httptest.NewRequest("POST", "/register", strings.NewReader(`{"tenant_name":"Acme","email":"owner@example.com","password":"correct-password"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 201 {
		t.Fatalf("register status = %d, body = %s", res.Code, res.Body.String())
	}
	if len(res.Result().Cookies()) != 1 || res.Result().Cookies()[0].Name != "agentscope_session" {
		t.Fatal("registration must set session cookie")
	}
}

func (f *fakeUserRepository) _contextCheck(_ context.Context) {}
