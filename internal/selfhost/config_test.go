package selfhost

import (
	"strings"
	"testing"
)

func TestParseConfigDerivesLocalOriginAndGeneratesSecrets(t *testing.T) {
	config, err := ParseConfig(strings.NewReader("WEB_PORT=3300\nAPI_PORT=8080\nMYSQL_PASSWORD=local\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if config.Values["WEB_ORIGIN"] != "http://127.0.0.1:3300" {
		t.Fatalf("origin = %q", config.Values["WEB_ORIGIN"])
	}
	if config.Values["SESSION_SECRET"] == "" || config.Values["AGENT_SIGNING_ENCRYPTION_KEY"] == "" {
		t.Fatal("expected generated secrets")
	}
}

func TestParseConfigRejectsUnsafeInput(t *testing.T) {
	for _, input := range []string{"UNKNOWN=value\n", "WEB_PORT=8080\nWEB_PORT=8081\n", "WEB_PORT=$HOME\n", "WEB_PORT=70000\n"} {
		if _, err := ParseConfig(strings.NewReader(input), false); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func TestParseConfigValidatesRedisSecuritySettings(t *testing.T) {
	config, err := ParseConfig(strings.NewReader("REDIS_ADDR=cache.example.com:6380\nREDIS_PASSWORD=secret\nREDIS_TLS_ENABLED=true\n"), false)
	if err != nil {
		t.Fatal(err)
	}
	if config.Values["REDIS_TLS_SERVER_NAME"] != "cache.example.com" {
		t.Fatalf("Redis TLS name = %q", config.Values["REDIS_TLS_SERVER_NAME"])
	}

	for _, input := range []string{"REDIS_USERNAME=cache-user\n", "REDIS_TLS_ENABLED=maybe\n", "REDIS_TLS_ENABLED=true\nREDIS_ADDR=redis\n"} {
		if _, err := ParseConfig(strings.NewReader(input), false); err == nil {
			t.Fatalf("accepted invalid Redis configuration %q", input)
		}
	}
}
