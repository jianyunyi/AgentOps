package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	MySQLDSN      string
	RedisAddr     string
	HTTPAddr      string
	SessionSecret string
	WebOrigin     string
	MaxBodyBytes  int64
	LLMBaseURL    string
	LLMAPIKey     string
	LLMModel      string
}

func Load() (Config, error) {
	cfg := Config{
		MySQLDSN:      os.Getenv("MYSQL_DSN"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		HTTPAddr:      os.Getenv("HTTP_ADDR"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		WebOrigin:     os.Getenv("WEB_ORIGIN"),
		LLMBaseURL:    os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:     os.Getenv("LLM_API_KEY"),
		LLMModel:      os.Getenv("LLM_MODEL"),
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
	return cfg, nil
}
