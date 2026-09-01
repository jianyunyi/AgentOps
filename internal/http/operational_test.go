package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apphttp "agentscope/internal/http"
	"github.com/gin-gonic/gin"
)

func TestHealthRoutesExposeLiveAndReadyStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apphttp.RegisterHealthRoutes(router, func(context.Context) error { return nil })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("live status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d", response.Code)
	}
}

func TestHealthReadyFailsWhenDependencyFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apphttp.RegisterHealthRoutes(router, func(context.Context) error { return errors.New("mysql down") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d", response.Code)
	}
}
