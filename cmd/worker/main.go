package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentscope/internal/outbox"
	"agentscope/internal/platform/config"
	"agentscope/internal/platform/database"
	platformmetrics "agentscope/internal/platform/metrics"
	platformredis "agentscope/internal/platform/redis"
	"agentscope/internal/policy"
	"agentscope/internal/risk"
	"agentscope/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(context.Background(), cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	llm := risk.StructuredAnalyzer(nil)
	if cfg.LLMBaseURL != "" {
		llm = &risk.OpenAICompatibleClient{BaseURL: cfg.LLMBaseURL, APIKey: cfg.LLMAPIKey, Model: cfg.LLMModel}
		healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := llm.(*risk.OpenAICompatibleClient).Health(healthCtx); err != nil {
			log.Printf("local llm health check failed; rules fallback remains active: %v", err)
		}
		healthCancel()
	}
	policyService := policy.NewService(policy.NewGORMRepository(db))
	riskService := risk.NewServiceWithPolicy(risk.NewGORMRepository(db), llm, policyService)
	client := platformredis.NewClient(cfg.RedisAddr)
	consumer := worker.NewConfiguredStreamConsumer(client, cfg.WorkerConsumerID, time.Duration(cfg.WorkerPendingIdleSeconds)*time.Second)
	processor := worker.NewAnalysisProcessor(cfg.WorkerMaxAttempts, func(ctx context.Context, message worker.AnalysisMessage) error {
		_, err := riskService.AnalyzeAndPersist(ctx, message.TenantID, message.TraceID, message.SpanID, message.Input)
		return err
	})
	publisher := outbox.NewPublisher(outbox.NewGORMRepository(db), func(ctx context.Context, event outbox.Event) error {
		var message worker.AnalysisMessage
		if err := json.Unmarshal(event.Payload, &message); err != nil {
			return err
		}
		if message.Version == "" {
			message.Version = "v1"
		}
		return worker.NewStreamPublisher(client).Publish(ctx, message)
	})
	workerMetrics := platformmetrics.NewWorker()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	outboxErrors := make(chan error, 1)
	go func() {
		outboxErrors <- worker.RunOutboxPublisherWithOptions(ctx, publisher, 500*time.Millisecond, worker.RunOptions{Metrics: workerMetrics})
	}()
	analysisErrors := make(chan error, 1)
	go func() {
		analysisErrors <- worker.RunWithOptions(ctx, consumer, processor, worker.RunOptions{Metrics: workerMetrics})
	}()
	metricsErrors := make(chan error, 1)
	go func() { metricsErrors <- worker.ServeMetrics(ctx, cfg.WorkerMetricsAddr, workerMetrics) }()
	select {
	case err := <-outboxErrors:
		cancel()
		if err != nil && err != context.Canceled {
			log.Fatal(err)
		}
	case err := <-analysisErrors:
		cancel()
		if err != nil && err != context.Canceled {
			log.Fatal(err)
		}
	case err := <-metricsErrors:
		cancel()
		if err != nil && err != context.Canceled {
			log.Fatal(err)
		}
	}
}
