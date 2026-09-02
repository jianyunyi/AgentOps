package config

import "testing"

func TestLoadRejectsMissingMySQLDSN(t *testing.T) {
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("SESSION_SECRET", "12345678901234567890123456789012")

	if _, err := Load(); err == nil {
		t.Fatal("Load() should reject a missing MYSQL_DSN")
	}
}

func TestLoadReadsRequiredConfiguration(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/agentscope?parseTime=true")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("SESSION_SECRET", "12345678901234567890123456789012")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.MySQLDSN == "" || got.RedisAddr == "" || got.HTTPAddr == "" || got.SessionSecret == "" {
		t.Fatalf("Load() returned incomplete config: %+v", got)
	}
}

func TestLoadUsesReplayProtectionDefaults(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("AGENT_REPLAY_WINDOW_SECONDS", "")
	t.Setenv("AGENT_NONCE_TTL_SECONDS", "")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentReplayWindow != 300 || got.AgentNonceTTL != 600 {
		t.Fatalf("replay defaults = window:%d ttl:%d", got.AgentReplayWindow, got.AgentNonceTTL)
	}
}

func TestLoadRejectsReplayConfigurationOutsideBounds(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("AGENT_REPLAY_WINDOW_SECONDS", "29")
	if _, err := Load(); err == nil {
		t.Fatal("replay window below 30 seconds must be rejected")
	}

	t.Setenv("AGENT_REPLAY_WINDOW_SECONDS", "300")
	t.Setenv("AGENT_NONCE_TTL_SECONDS", "3601")
	if _, err := Load(); err == nil {
		t.Fatal("nonce TTL above 3600 seconds must be rejected")
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/agentscope?parseTime=true")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("SESSION_SECRET", "12345678901234567890123456789012")
}
