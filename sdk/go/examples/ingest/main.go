package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/jianyunyi/AgentOps/sdk/go/agentops"
)

func main() {
	client, err := agentops.NewClient(agentops.Config{
		BaseURL:       os.Getenv("AGENTOPS_BASE_URL"),
		APIKey:        os.Getenv("AGENT_API_KEY"),
		SigningSecret: os.Getenv("AGENT_SIGNING_SECRET"),
	})
	if err != nil {
		log.Fatal("configure AgentScope SDK: ", err)
	}
	result, err := client.Ingest(context.Background(), agentops.Event{
		EventID:    "example-event-id",
		TraceID:    "example-trace-id",
		SpanID:     "example-span-id",
		EventType:  "llm_call",
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"prompt":"hello"}`),
	})
	if err != nil {
		log.Fatal("ingest AgentScope event: ", err)
	}
	log.Printf("event accepted; duplicate=%t", result.Duplicate)
}
