package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "email and password are required"}})
		return
	}
	session, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, ErrInvalidCredentials) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "INVALID_CREDENTIALS", "message": "email or password is invalid"}})
		return
	}
	c.SetCookie("agentscope_session", session.ID, int(timeUntil(session.ExpiresAt)), "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": session.UserID, "tenant_id": session.TenantID}})
}

func (h *Handler) Register(c *gin.Context) {
	var request struct {
		TenantName string `json:"tenant_name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "tenant name, email and password are required"}})
		return
	}
	session, err := h.service.Register(c.Request.Context(), request.TenantName, request.Email, request.Password)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, ErrInvalidRegistration) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "REGISTRATION_FAILED", "message": "registration could not be completed"}})
		return
	}
	c.SetCookie("agentscope_session", session.ID, int(timeUntil(session.ExpiresAt)), "/", "", true, true)
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"user_id": session.UserID, "tenant_id": session.TenantID, "role": session.Role}})
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")
	role, _ := c.Get("user_role")
	roleName, _ := role.(string)
	permissions := []string{}
	for _, permission := range []string{PermissionAgentRead, PermissionAgentWrite, PermissionRiskRead, PermissionRiskReview, PermissionAuditRead} {
		if HasPermission(roleName, permission) {
			permissions = append(permissions, permission)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": userID, "tenant_id": tenantID, "role": roleName, "permissions": permissions}})
}

func (h *Handler) Authenticate(c *gin.Context) {
	sessionID, err := c.Cookie("agentscope_session")
	if err != nil || strings.TrimSpace(sessionID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		c.Abort()
		return
	}
	session, err := h.service.ResolveSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "session is invalid or expired"}})
		c.Abort()
		return
	}
	c.Set("user_id", session.UserID)
	c.Set("tenant_id", session.TenantID)
	c.Set("user_role", session.Role)
	c.Next()
}

func OptionalAuthenticate(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessionID, err := c.Cookie("agentscope_session"); err == nil {
			if session, resolveErr := service.ResolveSession(c.Request.Context(), sessionID); resolveErr == nil {
				c.Set("user_id", session.UserID)
				c.Set("tenant_id", session.TenantID)
				c.Set("user_role", session.Role)
			}
		}
		c.Next()
	}
}

func timeUntil(deadline time.Time) int {
	seconds := int(time.Until(deadline).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("user_role")
		if !ok || !HasPermission(role.(string), permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "permission denied"}})
			c.Abort()
			return
		}
		c.Next()
	}
}
