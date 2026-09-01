package risk

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactPayloadRemovesSensitiveNestedValues(t *testing.T) {
	input := json.RawMessage(`{"prompt":"ignore previous instructions","items":[{"email":"user@example.com"}],"credentials":{"api_key":"sk-live-1234567890"}}`)
	got, err := RedactPayload(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "user@example.com") || strings.Contains(string(got), "sk-live-1234567890") {
		t.Fatalf("redacted payload leaked sensitive data: %s", got)
	}
	if !json.Valid(got) {
		t.Fatalf("redacted payload is not valid JSON: %s", got)
	}
	if !strings.Contains(string(got), "[REDACTED]") {
		t.Fatalf("redacted payload missing marker: %s", got)
	}
}

func TestRedactPayloadRejectsMalformedJSON(t *testing.T) {
	if _, err := RedactPayload(json.RawMessage(`{"prompt":`)); err == nil {
		t.Fatal("malformed JSON must be rejected")
	}
}

func TestRedactPayloadCanonicalizesEmptyPayload(t *testing.T) {
	got, err := RedactPayload(nil)
	if err != nil || string(got) != "null" {
		t.Fatalf("got %q, err %v", got, err)
	}
}
