package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	MySQLDSN                 string
	RedisAddr                string
	HTTPAddr                 string
	SessionSecret            string
	WebOrigin                string
	MaxBodyBytes             int64
	AgentReplayWindow        int64
	AgentNonceTTL            int64
	LLMBaseURL               string
	LLMAPIKey                string
	LLMModel                 string
	OIDCIssuerURL            string
	OIDCClientID             string
	OIDCClientSecret         string
	OIDCRedirectURL          string
	OIDCTenantID             string
	OIDCDefaultRole          string
	DBMaxOpenConns           int
	DBMaxIdleConns           int
	DBConnMaxLifetimeMinutes int
}

func Load() (Config, error) {
	cfg := Config{
		MySQLDSN:         os.Getenv("MYSQL_DSN"),
		RedisAddr:        os.Getenv("REDIS_ADDR"),
		HTTPAddr:         os.Getenv("HTTP_ADDR"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		WebOrigin:        os.Getenv("WEB_ORIGIN"),
		LLMBaseURL:       os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         os.Getenv("LLM_MODEL"),
		OIDCIssuerURL:    os.Getenv("OIDC_ISSUER_URL"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		OIDCTenantID:     os.Getenv("OIDC_TENANT_ID"),
		OIDCDefaultRole:  os.Getenv("OIDC_DEFAULT_ROLE"),
	}
	if cfg.MySQLDSN == "" {
		return Config{}, errors.New("MYSQL_DSN is required")
	}
	if cfg.RedisAddr == "" {
		return Config{}, errors.New("REDIS_ADDR is required")
	}
	if cfg.HTTPAddr == "" {
		return Config{}, errors.New("HTTP_ADDR is required")
	}
	if cfg.SessionSecret == "" {
		return Config{}, errors.New("SESSION_SECRET is required")
	}
	cfg.MaxBodyBytes = 2 * 1024 * 1024
	if raw := os.Getenv("MAX_BODY_BYTES"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1024 {
			return Config{}, errors.New("MAX_BODY_BYTES must be at least 1024")
		}
		cfg.MaxBodyBytes = value
	}
	replayWindow, parseErr := readBoundedSeconds("AGENT_REPLAY_WINDOW_SECONDS", 300, 30, 900)
	if parseErr != nil {
		return Config{}, parseErr
	}
	cfg.AgentReplayWindow = replayWindow
	nonceTTL, parseErr := readBoundedSeconds("AGENT_NONCE_TTL_SECONDS", 600, 60, 3600)
	if parseErr != nil {
		return Config{}, parseErr
	}
	cfg.AgentNonceTTL = nonceTTL
	cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetimeMinutes = 50, 10, 30
	if raw := os.Getenv("DB_MAX_OPEN_CONNS"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return Config{}, errors.New("DB_MAX_OPEN_CONNS must be positive")
		}
		cfg.DBMaxOpenConns = value
	}
	if raw := os.Getenv("DB_MAX_IDLE_CONNS"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return Config{}, errors.New("DB_MAX_IDLE_CONNS must be non-negative")
		}
		cfg.DBMaxIdleConns = value
	}
	if raw := os.Getenv("DB_CONN_MAX_LIFETIME_MINUTES"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return Config{}, errors.New("DB_CONN_MAX_LIFETIME_MINUTES must be positive")
		}
		cfg.DBConnMaxLifetimeMinutes = value
	}
	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("SESSION_SECRET must be at least 32 characters")
	}
	if cfg.LLMBaseURL != "" && cfg.LLMModel == "" {
		return Config{}, errors.New("LLM_MODEL is required when LLM_BASE_URL is configured")
	}
	return cfg, nil
}

func readBoundedSeconds(name string, fallback, minimum, maximum int64) (int64, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < minimum || value > maximum {
		return 0, errors.New(name + " must be between " + strconv.FormatInt(minimum, 10) + " and " + strconv.FormatInt(maximum, 10) + " seconds")
	}
	return value, nil
}
