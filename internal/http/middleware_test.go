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
