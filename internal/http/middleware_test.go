package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apphttp "agentscope/internal/http"
	"github.com/gin-gonic/gin"
)

type fakeLimiter struct {
	allowed    bool
	retryAfter time.Duration
}

func (l fakeLimiter) Allow(context.Context, string, int64, time.Duration) (bool, time.Duration, error) {
	return l.allowed, l.retryAfter, nil
}

func TestRequestIDMiddlewarePropagatesOrGeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.RequestID())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "req_existing")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Header().Get("X-Request-ID") != "req_existing" {
		t.Fatalf("request id was not propagated")
	}
}

func TestBodyLimitRejectsOversizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.BodyLimit(4))
	router.POST("/", func(c *gin.Context) {
		var payload map[string]string
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "too large"})
			return
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"long"}`))
	request.Header.Set("content-type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", response.Code)
	}
}

func TestCORSMiddlewareAllowsConfiguredOriginAndPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.CORS("https://console.example.com"))
	router.OPTIONS("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://console.example.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "https://console.example.com" {
		t.Fatalf("unexpected CORS response: %d %v", response.Code, response.Header())
	}
}

func TestRateLimitMiddlewareReturnsTooManyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.RateLimit(fakeLimiter{allowed: false, retryAfter: 2 * time.Second}, func(*gin.Context) string { return "client" }, 1, time.Minute))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "2" {
		t.Fatalf("unexpected rate limit response: %d %v", response.Code, response.Header())
	}
}

func TestCSRFMiddlewareRejectsUnsafeCookieRequestWithoutMatchingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.CSRF())
	router.POST("/api/v1/agents", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil)
	request.AddCookie(&http.Cookie{Name: "agentscope_session", Value: "session"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestCSRFMiddlewareAllowsMatchingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apphttp.CSRF())
	router.POST("/api/v1/agents", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/api/v1/agents", nil)
	request.AddCookie(&http.Cookie{Name: "agentscope_session", Value: "session"})
	request.AddCookie(&http.Cookie{Name: "agentscope_csrf", Value: "csrf"})
	request.Header.Set("X-CSRF-Token", "csrf")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestClientIPIgnoresForwardedHeadersUnlessProxyIsTrusted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		want           string
	}{
		{name: "untrusted forwarder", remoteAddr: "203.0.113.10:4321", want: "203.0.113.10"},
		{name: "trusted proxy", trustedProxies: []string{"10.20.0.0/16"}, remoteAddr: "10.20.1.8:4321", want: "198.51.100.24"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			if err := router.SetTrustedProxies(test.trustedProxies); err != nil {
				t.Fatal(err)
			}
			router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("X-Forwarded-For", "198.51.100.24")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if got := strings.TrimSpace(response.Body.String()); got != test.want {
				t.Fatalf("client IP = %q, want %q", got, test.want)
			}
		})
	}
}
