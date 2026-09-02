package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" || len(requestID) > 128 || strings.ContainsAny(requestID, "\r\n") {
			requestID = newRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		slog.Default().Info("http_request", "request_id", c.GetString("request_id"), "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(started).Milliseconds(), "client_ip", c.ClientIP())
	}
}

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes < 1 {
		maxBytes = 2 * 1024 * 1024
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func CORS(allowedOrigin string) gin.HandlerFunc {
	allowedOrigin = strings.TrimRight(strings.TrimSpace(allowedOrigin), "/")
	return func(c *gin.Context) {
		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
		if allowedOrigin != "" && origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-CSRF-Token, X-Agent-Timestamp, X-Agent-Nonce")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if cookie, err := c.Request.Cookie("agentscope_csrf"); err == nil {
			token = cookie.Value
		}
		if token == "" {
			token = newCSRFToken()
			c.Writer.Header().Add("Set-Cookie", csrfCookie(token).String())
		}
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete {
			path := c.Request.URL.Path
			exempt := path == "/api/v1/auth/login" || path == "/api/v1/auth/register" || path == "/api/v1/ingest/events"
			_, sessionErr := c.Request.Cookie("agentscope_session")
			hasSession := sessionErr == nil
			if hasSession && !exempt && c.GetHeader("X-CSRF-Token") != token {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "CSRF_FAILED", "message": "csrf token is invalid"}})
				return
			}
		}
		c.Next()
	}
}

func csrfCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "agentscope_csrf", Value: token, Path: "/", MaxAge: 86400, HttpOnly: false, Secure: true, SameSite: http.SameSiteLaxMode}
}

func newCSRFToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "csrf-unavailable"
	}
	return hex.EncodeToString(buf)
}

type RateLimiter interface {
	Allow(context.Context, string, int64, time.Duration) (bool, time.Duration, error)
}

func RateLimit(limiter RateLimiter, key func(*gin.Context) string, limit int64, window time.Duration) gin.HandlerFunc {
	return RateLimitPolicy(limiter, key, func(*gin.Context) (int64, time.Duration) { return limit, window })
}

func RateLimitPolicy(limiter RateLimiter, key func(*gin.Context) string, policy func(*gin.Context) (int64, time.Duration)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/health/") || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		limit, window := policy(c)
		allowed, retryAfter, err := limiter.Allow(c.Request.Context(), key(c), limit, window)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "RATE_LIMIT_UNAVAILABLE", "message": "rate limiter unavailable"}})
			return
		}
		if !allowed {
			seconds := int(retryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "RATE_LIMITED", "message": "too many requests"}})
			return
		}
		c.Next()
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "req-unknown"
	}
	return "req_" + hex.EncodeToString(buf)
}
