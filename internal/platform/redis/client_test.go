package redis_test

import (
	"crypto/tls"
	"testing"

	platformredis "agentscope/internal/platform/redis"
)

func TestNewClientWithConfigAppliesAuthAndVerifiedTLS(t *testing.T) {
	client, err := platformredis.NewClientWithConfig(platformredis.ClientConfig{
		Addr:          "cache.example.com:6380",
		Username:      "cache-user",
		Password:      "cache-password",
		TLSEnabled:    true,
		TLSServerName: "redis.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	options := client.Options()
	if options.Username != "cache-user" || options.Password != "cache-password" {
		t.Fatalf("Redis authentication options = username:%q password-set:%t", options.Username, options.Password != "")
	}
	if options.TLSConfig == nil || options.TLSConfig.MinVersion != tls.VersionTLS12 || options.TLSConfig.ServerName != "redis.example.com" || options.TLSConfig.InsecureSkipVerify {
		t.Fatalf("Redis TLS config is not strict: %+v", options.TLSConfig)
	}
}

func TestNewClientWithConfigInfersTLSNameOrRejectsInvalidAddress(t *testing.T) {
	client, err := platformredis.NewClientWithConfig(platformredis.ClientConfig{Addr: "redis.internal:6380", TLSEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := client.Options().TLSConfig.ServerName; got != "redis.internal" {
		t.Fatalf("inferred TLS server name = %q", got)
	}
	_ = client.Close()

	if _, err := platformredis.NewClientWithConfig(platformredis.ClientConfig{Addr: "redis", TLSEnabled: true}); err == nil {
		t.Fatal("TLS without an explicit or inferable server name must fail")
	}
}
