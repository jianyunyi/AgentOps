package http

import (
	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	"agentscope/internal/policy"
	"agentscope/internal/risk"
	"agentscope/internal/trace"
	"github.com/gin-gonic/gin"
)

func NewRouter(traceService *trace.Service, traceQuery trace.QueryRepository, authenticator trace.Authenticator) *gin.Engine {
	return trace.NewRouter(traceService, traceQuery, authenticator)
}

func NewApplicationRouter(authService *auth.Service, agentService *agent.Service, auditService *audit.Service, riskService *risk.Service, traceService *trace.Service, traceQuery trace.QueryRepository, policyService *policy.Service, middleware ...gin.HandlerFunc) *gin.Engine {
	router := trace.NewRouter(traceService, traceQuery, agentService)
	router.Use(middleware...)
	router.Use(auth.OptionalAuthenticate(authService))
	authHandler := auth.NewHandler(authService)
	agentHandler := agent.NewHandler(agentService)
	auditHandler := audit.NewHandler(auditService)
	riskHandler := risk.NewHandler(riskService)
	policyHandler := policy.NewHandler(policyService)
	console := router.Group("/api/v1")
	console.Use(authHandler.Authenticate)
	console.GET("/auth/me", authHandler.Me)
	console.POST("/agents", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.Create)
	console.GET("/agents", auth.RequirePermission(auth.PermissionAgentRead), agentHandler.List)
	console.POST("/agents/:id/rotate-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RotateKey)
	console.POST("/agents/:id/revoke-key", auth.RequirePermission(auth.PermissionAgentWrite), agentHandler.RevokeKey)
	console.GET("/audit-logs", auth.RequirePermission(auth.PermissionAuditRead), auditHandler.List)
	console.GET("/members", auth.RequirePermission(auth.PermissionMemberRead), authHandler.ListMembers)
	console.GET("/members/invitations", auth.RequirePermission(auth.PermissionMemberRead), authHandler.ListInvitations)
	console.POST("/members/invitations", auth.RequirePermission(auth.PermissionMemberWrite), authHandler.CreateInvitation)
	console.PATCH("/members/:id/role", auth.RequirePermission(auth.PermissionMemberWrite), authHandler.ChangeMemberRole)
	console.POST("/members/:id/disable", auth.RequirePermission(auth.PermissionMemberWrite), authHandler.DisableMember)
	console.POST("/members/:id/transfer-owner", auth.RequirePermission(auth.PermissionMemberWrite), authHandler.TransferOwner)
	console.GET("/risk-events", auth.RequirePermission(auth.PermissionRiskRead), riskHandler.List)
	console.POST("/risk-events/:id/review", auth.RequirePermission(auth.PermissionRiskReview), riskHandler.Review)
	console.GET("/policies", auth.RequirePermission(auth.PermissionPolicyRead), policyHandler.List)
	console.POST("/policies", auth.RequirePermission(auth.PermissionPolicyWrite), policyHandler.Create)
	console.POST("/policies/:id/activate", auth.RequirePermission(auth.PermissionPolicyWrite), policyHandler.Activate)
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.POST("/api/v1/auth/register", authHandler.Register)
	router.POST("/api/v1/auth/logout", authHandler.Logout)
	router.POST("/api/v1/auth/invitations/accept", authHandler.AcceptInvitation)
	return router
}
