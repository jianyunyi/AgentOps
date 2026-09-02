package agentops

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewClientConfig(t *testing.T) {
	secret := base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	valid := Config{BaseURL: "https://agentscope.example.com/", APIKey: "ag_live_test", SigningSecret: secret}
	client, err := NewClient(valid)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.baseURL.String() != "https://agentscope.example.com" {
		t.Fatalf("base URL = %q", client.baseURL)
	}
	if client.httpClient.Timeout != 10*time.Second || client.retry.MaxAttempts != 3 {
		t.Fatalf("defaults = timeout:%s attempts:%d", client.httpClient.Timeout, client.retry.MaxAttempts)
	}
}

func TestNewClientConfigRejectsInvalidValues(t *testing.T) {
	secret := base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "missing url", cfg: Config{APIKey: "ag_live_test", SigningSecret: secret}},
		{name: "relative url", cfg: Config{BaseURL: "/agentscope", APIKey: "ag_live_test", SigningSecret: secret}},
		{name: "unsupported scheme", cfg: Config{BaseURL: "ftp://agentscope.example.com", APIKey: "ag_live_test", SigningSecret: secret}},
		{name: "missing host", cfg: Config{BaseURL: "https:/agentscope.example.com", APIKey: "ag_live_test", SigningSecret: secret}},
		{name: "missing api key", cfg: Config{BaseURL: "https://agentscope.example.com", SigningSecret: secret}},
		{name: "missing signing secret", cfg: Config{BaseURL: "https://agentscope.example.com", APIKey: "ag_live_test"}},
		{name: "invalid signing secret", cfg: Config{BaseURL: "https://agentscope.example.com", APIKey: "ag_live_test", SigningSecret: "not-a-secret"}},
		{name: "too few attempts", cfg: Config{BaseURL: "https://agentscope.example.com", APIKey: "ag_live_test", SigningSecret: secret, Retry: RetryPolicy{MaxAttempts: -1}}},
		{name: "too many attempts", cfg: Config{BaseURL: "https://agentscope.example.com", APIKey: "ag_live_test", SigningSecret: secret, Retry: RetryPolicy{MaxAttempts: 6}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewClient(tt.cfg); err == nil {
				t.Fatal("NewClient() should reject invalid configuration")
			}
		})
	}
}

func TestNewClientPreservesInjectedHTTPClient(t *testing.T) {
	secret := base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	httpClient := &http.Client{Timeout: 3 * time.Second}
	client, err := NewClient(Config{BaseURL: "https://agentscope.example.com", APIKey: "ag_live_test", SigningSecret: secret, HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	if client.httpClient != httpClient || client.httpClient.Timeout != 3*time.Second {
		t.Fatal("injected HTTP client must not be replaced or mutated")
	}
	if strings.Contains(client.baseURL.String(), "ag_live_test") {
		t.Fatal("credentials must not be included in the base URL")
	}
}
