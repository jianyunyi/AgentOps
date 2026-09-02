package agent

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Environment string `json:"environment"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "agent name is required"}})
		return
	}
	result, err := h.service.CreateAgent(c.Request.Context(), CreateAgentInput{TenantID: tenantID.(string), Name: request.Name, Description: request.Description, Environment: request.Environment})
	if err != nil {
		if errors.Is(err, ErrSigningSecretUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AGENT_AUTH_UNAVAILABLE", "message": "agent credential signing is temporarily unavailable"}})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "AGENT_CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"agent": result.Agent, "api_key": result.RawAPIKey, "signing_secret": result.RawSigningSecret}})
}

func (h *Handler) List(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	agents, err := h.service.repo.ListAgents(c.Request.Context(), tenantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to query agents"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": agents})
}

func (h *Handler) MigrationStatus(c *gin.Context) {
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	page, pageSize, err := migrationPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "invalid migration pagination"}})
		return
	}
	status, err := h.service.GetCredentialMigrationStatus(c.Request.Context(), tenantIDValue.(string), c.Query("q"), page, pageSize)
	if err != nil {
		if errors.Is(err, ErrMigrationStatusUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "MIGRATION_STATUS_UNAVAILABLE", "message": "agent credential migration status is temporarily unavailable"}})
			return
		}
		if errors.Is(err, ErrInvalidMigrationQuery) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "invalid migration query"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "MIGRATION_STATUS_FAILED", "message": "failed to query credential migration status"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

func migrationPagination(c *gin.Context) (int, int, error) {
	page, pageSize := 1, defaultMigrationPageSize
	var err error
	if raw := c.Query("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, ErrInvalidMigrationQuery
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, ErrInvalidMigrationQuery
		}
	}
	if page < 1 || pageSize < 1 || pageSize > maxMigrationPageSize {
		return 0, 0, ErrInvalidMigrationQuery
	}
	return page, pageSize, nil
}

func (h *Handler) RotateKey(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	result, err := h.service.RotateAPIKey(c.Request.Context(), tenantID.(string), c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrSigningSecretUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AGENT_AUTH_UNAVAILABLE", "message": "agent credential signing is temporarily unavailable"}})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "KEY_ROTATION_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"agent": result.Agent, "api_key": result.RawAPIKey, "signing_secret": result.RawSigningSecret}})
}

func (h *Handler) RevokeKey(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	if err := h.service.RevokeAPIKey(c.Request.Context(), tenantID.(string), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "KEY_REVOCATION_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
