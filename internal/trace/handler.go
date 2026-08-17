package trace

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"agentscope/internal/agent"
	"github.com/gin-gonic/gin"
)

type Authenticator interface {
	AuthenticateAPIKey(ctx context.Context, rawKey string) (agent.Identity, error)
}

type Handler struct {
	service       *Service
	query         QueryRepository
	authenticator Authenticator
}

func NewRouter(service *Service, query QueryRepository, authenticator Authenticator) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	handler := &Handler{service: service, query: query, authenticator: authenticator}
	router.POST("/api/v1/ingest/events", handler.ingest)
	router.GET("/api/v1/traces", handler.listTraces)
	router.GET("/api/v1/traces/:traceId", handler.getTrace)
	router.GET("/api/v1/traces/:traceId/spans", handler.listSpans)
	return router
}

func (h *Handler) ingest(c *gin.Context) {
	identity, ok := h.authenticate(c)
	if !ok {
		return
	}
	var event Event
	if err := c.ShouldBindJSON(&event); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_EVENT", "request body is invalid")
		return
	}
	result, err := h.service.Ingest(c.Request.Context(), IngestContext{TenantID: identity.TenantID, AgentID: identity.AgentID}, event)
	if err != nil {
		writeIngestError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"duplicate": result.Duplicate}})
}

func (h *Handler) listTraces(c *gin.Context) {
	identity, ok := h.authenticate(c)
	if !ok {
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	traces, total, err := h.query.ListTraces(c.Request.Context(), identity.TenantID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query traces")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": traces, "pagination": gin.H{"page": page, "page_size": pageSize, "total": total}})
}

func (h *Handler) getTrace(c *gin.Context) {
	identity, ok := h.authenticate(c)
	if !ok {
		return
	}
	trace, err := h.query.FindTrace(c.Request.Context(), identity.TenantID, c.Param("traceId"))
	if errors.Is(err, ErrTraceNotFound) {
		writeError(c, http.StatusNotFound, "TRACE_NOT_FOUND", "trace does not exist")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query trace")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trace})
}

func (h *Handler) listSpans(c *gin.Context) {
	identity, ok := h.authenticate(c)
	if !ok {
		return
	}
	spans, err := h.query.ListSpans(c.Request.Context(), identity.TenantID, c.Param("traceId"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query spans")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": spans})
}

func (h *Handler) authenticate(c *gin.Context) (agent.Identity, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED_AGENT", "agent authorization is required")
		return agent.Identity{}, false
	}
	identity, err := h.authenticator.AuthenticateAPIKey(c.Request.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED_AGENT", "agent authorization is invalid")
		return agent.Identity{}, false
	}
	return identity, true
}

func writeIngestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidEventType), errors.Is(err, ErrEventTimeInvalid), errors.Is(err, ErrPayloadTooLarge):
		writeError(c, http.StatusBadRequest, "INVALID_EVENT", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to ingest event")
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
