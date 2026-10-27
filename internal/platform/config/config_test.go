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
	if got.AgentSignatureRequired {
		t.Fatal("agent signature should be optional by default")
	}
	if len(got.TrustedProxyCIDRs) != 0 {
		t.Fatalf("trusted proxies must be empty by default: %+v", got.TrustedProxyCIDRs)
	}
}

func TestLoadReadsAndValidatesTrustedProxyCIDRs(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 2001:db8::/32")
	got, err := Load()
	if err != nil || len(got.TrustedProxyCIDRs) != 2 {
		t.Fatalf("trusted proxy config = %#v, err = %v", got.TrustedProxyCIDRs, err)
	}
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")
	if _, err := Load(); err == nil {
		t.Fatal("invalid trusted proxy CIDR must be rejected")
	}
}

func TestLoadReadsRedisAuthenticationAndTLS(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("REDIS_USERNAME", "cache-user")
	t.Setenv("REDIS_PASSWORD", "cache-password")
	t.Setenv("REDIS_TLS_ENABLED", "true")
	t.Setenv("REDIS_TLS_SERVER_NAME", "redis.example.com")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.RedisUsername != "cache-user" || got.RedisPassword != "cache-password" || !got.RedisTLSEnabled || got.RedisTLSServerName != "redis.example.com" {
		t.Fatalf("Redis security config = %+v", got)
	}
}

func TestLoadDerivesRedisTLSServerNameAndRejectsIncompleteAuth(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("REDIS_TLS_ENABLED", "true")
	got, err := Load()
	if err != nil || got.RedisTLSServerName != "127.0.0.1" {
		t.Fatalf("derived Redis TLS server name = %q, err = %v", got.RedisTLSServerName, err)
	}

	setRequiredEnvironment(t)
	t.Setenv("REDIS_USERNAME", "cache-user")
	if _, err := Load(); err == nil {
		t.Fatal("Redis username without a password must be rejected")
	}
}

func TestLoadReadsLegacySessionPurgeToggle(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("SESSION_LEGACY_SESSION_PURGE", "true")
	got, err := Load()
	if err != nil || !got.PurgeLegacySessions {
		t.Fatalf("legacy session purge = %t, err = %v", got.PurgeLegacySessions, err)
	}
	t.Setenv("SESSION_LEGACY_SESSION_PURGE", "not-a-bool")
	if _, err := Load(); err == nil {
		t.Fatal("invalid legacy session purge value must be rejected")
	}
}

func TestLoadReadsAgentSigningConfiguration(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("AGENT_SIGNING_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("AGENT_SIGNATURE_REQUIRED", "true")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentSigningEncryptionKey == "" || !got.AgentSignatureRequired {
		t.Fatalf("agent signing config = %+v", got)
	}
}

func TestLoadRejectsRequiredAgentSigningWithoutEncryptionKey(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("AGENT_SIGNING_ENCRYPTION_KEY", "")
	t.Setenv("AGENT_SIGNATURE_REQUIRED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("required agent signing without encryption key must be rejected")
	}
}

func TestLoadUsesReplayProtectionDefaults(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("AGENT_REPLAY_WINDOW_SECONDS", "")
	t.Setenv("AGENT_NONCE_TTL_SECONDS", "")
	t.Setenv("WORKER_CONSUMER_ID", "")
	t.Setenv("WORKER_PENDING_IDLE_SECONDS", "")
	t.Setenv("WORKER_MAX_ATTEMPTS", "")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentReplayWindow != 300 || got.AgentNonceTTL != 600 {
		t.Fatalf("replay defaults = window:%d ttl:%d", got.AgentReplayWindow, got.AgentNonceTTL)
	}
	if got.WorkerPendingIdleSeconds != 120 || got.WorkerMaxAttempts != 3 || got.WorkerConsumerID != "" || got.WorkerMetricsAddr != ":9091" {
		t.Fatalf("worker defaults = idle:%d attempts:%d consumer:%q metrics:%q", got.WorkerPendingIdleSeconds, got.WorkerMaxAttempts, got.WorkerConsumerID, got.WorkerMetricsAddr)
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

func TestLoadReadsWorkerConfiguration(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("WORKER_CONSUMER_ID", "worker-canary-01")
	t.Setenv("WORKER_PENDING_IDLE_SECONDS", "300")
	t.Setenv("WORKER_MAX_ATTEMPTS", "7")
	t.Setenv("WORKER_METRICS_ADDR", "127.0.0.1:19091")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkerConsumerID != "worker-canary-01" || got.WorkerPendingIdleSeconds != 300 || got.WorkerMaxAttempts != 7 || got.WorkerMetricsAddr != "127.0.0.1:19091" {
		t.Fatalf("worker config = %+v", got)
	}
}

func TestLoadRejectsWorkerConfigurationOutsideBounds(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("WORKER_PENDING_IDLE_SECONDS", "29")
	if _, err := Load(); err == nil {
		t.Fatal("worker pending idle below 30 seconds must be rejected")
	}

	t.Setenv("WORKER_PENDING_IDLE_SECONDS", "120")
	t.Setenv("WORKER_MAX_ATTEMPTS", "21")
	if _, err := Load(); err == nil {
		t.Fatal("worker max attempts above 20 must be rejected")
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/agentscope?parseTime=true")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("REDIS_USERNAME", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_TLS_ENABLED", "")
	t.Setenv("REDIS_TLS_SERVER_NAME", "")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("SESSION_SECRET", "12345678901234567890123456789012")
	t.Setenv("SESSION_LEGACY_SESSION_PURGE", "")
}
