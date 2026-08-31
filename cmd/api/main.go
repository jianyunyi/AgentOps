package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"agentscope/internal/agent"
	"agentscope/internal/auth"
	apihttp "agentscope/internal/http"
	"agentscope/internal/platform/config"
	"agentscope/internal/platform/database"
	platformredis "agentscope/internal/platform/redis"
	"agentscope/internal/trace"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	_ = platformredis.NewClient(cfg.RedisAddr)
	agentRepo := agent.NewGORMRepository(db)
	agentService := agent.NewService(agentRepo)
	authService := auth.NewService(auth.NewGORMRepository(db), cfg.SessionSecret)
	traceRepo := trace.NewGORMRepository(db)
	traceService := trace.NewService(traceRepo)
	router := apihttp.NewApplicationRouter(authService, agentService, traceService, traceRepo)

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router}
	log.Printf("agentscope api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
