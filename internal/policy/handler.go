package policy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	items, err := h.service.List(c.Request.Context(), tenantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to query policies"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}
func (h *Handler) Create(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	actorID, _ := c.Get("user_id")
	var request struct {
		Name          string `json:"name"`
		RulesEnabled  bool   `json:"rules_enabled"`
		LLMEnabled    bool   `json:"llm_enabled"`
		MaxInputBytes int    `json:"max_input_bytes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "policy fields are invalid"}})
		return
	}
	item, err := h.service.Create(c.Request.Context(), CreateInput{TenantID: tenantID.(string), CreatedBy: actorID.(string), Name: request.Name, RulesEnabled: request.RulesEnabled, LLMEnabled: request.LLMEnabled, MaxInputBytes: request.MaxInputBytes})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "POLICY_CREATE_FAILED", "message": "policy could not be created"}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}
func (h *Handler) Activate(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	if err := h.service.Activate(c.Request.Context(), tenantID.(string), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "POLICY_NOT_FOUND", "message": "policy not found"}})
		return
	}
	c.Status(http.StatusNoContent)
}
func ParseLimit(value string) int { n, _ := strconv.Atoi(value); return n }
