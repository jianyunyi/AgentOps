package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

type MemberResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

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
	setSessionCookie(c, session.ID, session.ExpiresAt)
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
	setSessionCookie(c, session.ID, session.ExpiresAt)
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"user_id": session.UserID, "tenant_id": session.TenantID, "role": session.Role}})
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")
	role, _ := c.Get("user_role")
	roleName, _ := role.(string)
	permissions := []string{}
	for _, permission := range []string{PermissionAgentRead, PermissionAgentWrite, PermissionRiskRead, PermissionRiskReview, PermissionAuditRead, PermissionMemberRead, PermissionMemberWrite} {
		if HasPermission(roleName, permission) {
			permissions = append(permissions, permission)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": userID, "tenant_id": tenantID, "role": roleName, "permissions": permissions}})
}

func setSessionCookie(c *gin.Context, sessionID string, expiresAt time.Time) {
	cookie := &http.Cookie{Name: "agentscope_session", Value: sessionID, Path: "/", MaxAge: timeUntil(expiresAt), HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
	c.Header("Set-Cookie", cookie.String())
}

func (h *Handler) Logout(c *gin.Context) {
	cookie := &http.Cookie{Name: "agentscope_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
	c.Header("Set-Cookie", cookie.String())
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListMembers(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	members, err := h.service.ListMembers(c.Request.Context(), tenantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to query members"}})
		return
	}
	result := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		result = append(result, MemberResponse{ID: member.ID, Email: member.Email, Role: member.Role, Status: member.Status, CreatedAt: member.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ChangeMemberRole(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	actorID, _ := c.Get("user_id")
	var request struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "role is required"}})
		return
	}
	if err := h.service.ChangeMemberRole(c.Request.Context(), tenantID.(string), actorID.(string), c.Param("id"), request.Role); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUserNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "MEMBER_UPDATE_FAILED", "message": "member role could not be updated"}})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DisableMember(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	actorID, _ := c.Get("user_id")
	if err := h.service.DisableMember(c.Request.Context(), tenantID.(string), actorID.(string), c.Param("id")); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUserNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "MEMBER_DISABLE_FAILED", "message": "member could not be disabled"}})
		return
	}
	c.Status(http.StatusNoContent)
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
