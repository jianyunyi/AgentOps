package http

import (
	"agentscope/internal/agent"
	"agentscope/internal/auth"
	"agentscope/internal/trace"
	"github.com/gin-gonic/gin"
)

func NewRouter(traceService *trace.Service, traceQuery trace.QueryRepository, authenticator trace.Authenticator) *gin.Engine {
	return trace.NewRouter(traceService, traceQuery, authenticator)
}

func NewApplicationRouter(authService *auth.Service, agentService *agent.Service, traceService *trace.Service, traceQuery trace.QueryRepository) *gin.Engine {
	router := trace.NewRouter(traceService, traceQuery, agentService)
	router.Use(auth.OptionalAuthenticate(authService))
	authHandler := auth.NewHandler(authService)
	agentHandler := agent.NewHandler(agentService)
	console := router.Group("/api/v1")
	console.Use(authHandler.Authenticate)
	console.POST("/agents", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.Create)
	console.GET("/agents", auth.RequirePermission(auth.PermissionAgentRead), agentHandler.List)
	console.POST("/agents/:id/rotate-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RotateKey)
	console.POST("/agents/:id/revoke-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RevokeKey)
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.POST("/api/v1/auth/register", authHandler.Register)
	return router
}
