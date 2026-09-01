package risk

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	events, total, err := h.service.List(c.Request.Context(), c.GetString("tenant_id"), (page-1)*pageSize, pageSize, c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to query risk events"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events, "pagination": gin.H{"page": page, "page_size": pageSize, "total": total}})
}
func (h *Handler) Review(c *gin.Context) {
	var request struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": "status is required"}})
		return
	}
	if err := h.service.Review(c.Request.Context(), c.GetString("tenant_id"), c.Param("id"), request.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RISK_REVIEW_FAILED", "message": err.Error()}})
		return
	}
	c.Status(http.StatusNoContent)
}
