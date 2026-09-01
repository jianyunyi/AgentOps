package http

import (
	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	"agentscope/internal/risk"
	"agentscope/internal/trace"
	"github.com/gin-gonic/gin"
)

func NewRouter(traceService *trace.Service, traceQuery trace.QueryRepository, authenticator trace.Authenticator) *gin.Engine {
	return trace.NewRouter(traceService, traceQuery, authenticator)
}

func NewApplicationRouter(authService *auth.Service, agentService *agent.Service, auditService *audit.Service, riskService *risk.Service, traceService *trace.Service, traceQuery trace.QueryRepository, middleware ...gin.HandlerFunc) *gin.Engine {
	router := trace.NewRouter(traceService, traceQuery, agentService)
	router.Use(middleware...)
	router.Use(auth.OptionalAuthenticate(authService))
	authHandler := auth.NewHandler(authService)
	agentHandler := agent.NewHandler(agentService)
	auditHandler := audit.NewHandler(auditService)
	riskHandler := risk.NewHandler(riskService)
	console := router.Group("/api/v1")
	console.Use(authHandler.Authenticate)
	console.GET("/auth/me", authHandler.Me)
	console.POST("/auth/logout", authHandler.Logout)
	console.POST("/agents", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.Create)
	console.GET("/agents", auth.RequirePermission(auth.PermissionAgentRead), agentHandler.List)
	console.POST("/agents/:id/rotate-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RotateKey)
	console.POST("/agents/:id/revoke-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RevokeKey)
	console.GET("/audit-logs", auth.RequirePermission(auth.PermissionAuditRead), auditHandler.List)
	console.GET("/risk-events", auth.RequirePermission(auth.PermissionRiskRead), riskHandler.List)
	console.POST("/risk-events/:id/review", auth.RequirePermission(auth.PermissionRiskReview), riskHandler.Review)
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.POST("/api/v1/auth/register", authHandler.Register)
	return router
}
