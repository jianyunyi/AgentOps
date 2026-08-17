package main

import (
	"context"
	"log"

	"agentscope/internal/platform/config"
	platformredis "agentscope/internal/platform/redis"
	"agentscope/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	client := platformredis.NewClient(cfg.RedisAddr)
	consumer := worker.NewStreamConsumer(client)
	processor := worker.NewAnalysisProcessor(3, func(context.Context, worker.AnalysisMessage) error {
		return nil
	})
	if err := worker.Run(context.Background(), consumer, processor); err != nil {
		log.Fatal(err)
	}
}
