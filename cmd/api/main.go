package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentscope/internal/agent"
	"agentscope/internal/audit"
	"agentscope/internal/auth"
	apihttp "agentscope/internal/http"
	"agentscope/internal/outbox"
	"agentscope/internal/platform/config"
	"agentscope/internal/platform/database"
	"agentscope/internal/platform/metrics"
	platformratelimit "agentscope/internal/platform/ratelimit"
	platformredis "agentscope/internal/platform/redis"
	"agentscope/internal/policy"
	"agentscope/internal/risk"
	"agentscope/internal/trace"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.OpenWithPool(ctx, cfg.MySQLDSN, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, time.Duration(cfg.DBConnMaxLifetimeMinutes)*time.Minute)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	redisClient := platformredis.NewClient(cfg.RedisAddr)
	agentRepo := agent.NewGORMRepository(db)
	auditService := audit.NewService(audit.NewGORMRepository(db))
	policyService := policy.NewService(policy.NewGORMRepository(db))
	riskService := risk.NewServiceWithPolicy(risk.NewGORMRepository(db), nil, policyService)
	agentService := agent.NewServiceWithAudit(agentRepo, auditService)
	authRepo := auth.NewGORMRepository(db)
	authService := auth.NewServiceWithAudit(authRepo, cfg.SessionSecret, auditService)
	var oidcService *auth.OIDCService
	if cfg.OIDCIssuerURL != "" {
		oidcService, err = auth.NewOIDCService(ctx, auth.OIDCConfig{IssuerURL: cfg.OIDCIssuerURL, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret, RedirectURL: cfg.OIDCRedirectURL, TenantID: cfg.OIDCTenantID, DefaultRole: cfg.OIDCDefaultRole, StateSecret: cfg.SessionSecret})
		if err != nil {
			log.Printf("oidc disabled: %v", err)
		}
	}
	traceRepo := trace.NewGORMRepository(db)
	outboxService := outbox.NewService(outbox.NewGORMRepository(db))
	traceService := trace.NewServiceWithOutbox(traceRepo, outboxService)
	rateLimiter := platformratelimit.New(platformratelimit.NewRedisStore(redisClient))
	requestMetrics := metrics.New()
	router := apihttp.NewApplicationRouter(authService, agentService, auditService, riskService, traceService, traceRepo, policyService,
		requestMetrics.Middleware(), apihttp.RequestID(), apihttp.RequestLogger(), apihttp.BodyLimit(cfg.MaxBodyBytes), apihttp.CORS(cfg.WebOrigin), apihttp.CSRF(),
		apihttp.RateLimitPolicy(rateLimiter, func(c *gin.Context) string { return c.ClientIP() + ":" + c.Request.Method + ":" + c.Request.URL.Path }, func(c *gin.Context) (int64, time.Duration) {
			if c.Request.URL.Path == "/api/v1/auth/login" || c.Request.URL.Path == "/api/v1/auth/register" {
				return 10, time.Minute
			}
			if c.Request.URL.Path == "/api/v1/ingest/events" {
				return 600, time.Minute
			}
			return 120, time.Minute
		}))
	authHandler := auth.NewHandlerWithOIDC(authService, oidcService)
	router.GET("/api/v1/auth/oidc/login", authHandler.OIDCLogin)
	router.GET("/api/v1/auth/oidc/callback", authHandler.OIDCCallback)
	router.GET("/metrics", gin.WrapF(requestMetrics.Handler()))
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	apihttp.RegisterHealthRoutes(router, func(ctx context.Context) error {
		if err := sqlDB.PingContext(ctx); err != nil {
			return err
		}
		return redisClient.Ping(ctx).Err()
	})

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 120 * time.Second, MaxHeaderBytes: 32 * 1024}
	log.Printf("agentscope api listening on %s", cfg.HTTPAddr)
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()
	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	case <-stop.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		<-serverErr
	}
}
