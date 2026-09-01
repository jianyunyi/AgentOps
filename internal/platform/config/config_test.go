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
