package agentops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jianyunyi/AgentOps/sdk/go/agentops"
)

func TestSDKIngestAgainstRunningAPI(t *testing.T) {
	if os.Getenv("AGENTSCOPE_INTEGRATION") != "1" {
		t.Skip("set AGENTSCOPE_INTEGRATION=1 to run SDK API integration tests")
	}
	baseURL, apiKey, signingSecret := os.Getenv("AGENTOPS_SDK_BASE_URL"), os.Getenv("AGENT_API_KEY"), os.Getenv("AGENT_SIGNING_SECRET")
	if baseURL == "" || apiKey == "" || signingSecret == "" {
		t.Skip("set AGENTOPS_SDK_BASE_URL, AGENT_API_KEY, and AGENT_SIGNING_SECRET to run SDK API integration tests")
	}
	client, err := agentops.NewClient(agentops.Config{BaseURL: baseURL, APIKey: apiKey, SigningSecret: signingSecret})
	if err != nil {
		t.Fatal(err)
	}
	eventID := fmt.Sprintf("sdk_it_%d", time.Now().UnixNano())
	result, err := client.Ingest(context.Background(), agentops.Event{
		EventID:    eventID,
		TraceID:    eventID + "_trace",
		SpanID:     eventID + "_span",
		EventType:  "llm_call",
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"input":"sdk integration"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Duplicate {
		t.Fatal("first SDK integration event must not be a duplicate")
	}
}
