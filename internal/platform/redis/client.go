package redis

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"

	redisv9 "github.com/redis/go-redis/v9"
)

const AnalysisStream = "agentscope:analysis:v1"

type ClientConfig struct {
	Addr          string
	Username      string
	Password      string
	TLSEnabled    bool
	TLSServerName string
}

func NewClient(addr string) *redisv9.Client {
	client, _ := NewClientWithConfig(ClientConfig{Addr: addr})
	return client
}

func NewClientWithConfig(cfg ClientConfig) (*redisv9.Client, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, errors.New("REDIS_ADDR is required")
	}
	options := &redisv9.Options{Addr: cfg.Addr, Username: cfg.Username, Password: cfg.Password}
	if cfg.TLSEnabled {
		serverName := strings.TrimSpace(cfg.TLSServerName)
		if serverName == "" {
			host, _, err := net.SplitHostPort(cfg.Addr)
			if err != nil || host == "" {
				return nil, errors.New("REDIS_TLS_SERVER_NAME is required when it cannot be inferred from REDIS_ADDR")
			}
			serverName = host
		}
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	}
	return redisv9.NewClient(options), nil
}
