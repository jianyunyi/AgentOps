package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ReadinessCheck func(context.Context) error

func RegisterHealthRoutes(router *gin.Engine, check ReadinessCheck) {
	router.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "alive"}) })
	router.GET("/health/ready", func(c *gin.Context) {
		if check != nil {
			if err := check(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}
