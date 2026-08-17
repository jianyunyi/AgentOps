package http

import (
	"agentscope/internal/trace"
	"github.com/gin-gonic/gin"
)

func NewRouter(traceService *trace.Service, traceQuery trace.QueryRepository, authenticator trace.Authenticator) *gin.Engine {
	return trace.NewRouter(traceService, traceQuery, authenticator)
}
