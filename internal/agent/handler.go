package agent

import (
	"net/http"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "AGENT_CREATE_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"agent": result.Agent, "api_key": result.RawAPIKey}})
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

func (h *Handler) RotateKey(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "login is required"}})
		return
	}
	result, err := h.service.RotateAPIKey(c.Request.Context(), tenantID.(string), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "KEY_ROTATION_FAILED", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"agent": result.Agent, "api_key": result.RawAPIKey}})
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
