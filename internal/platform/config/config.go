package config

import (
	"errors"
	"os"
)

type Config struct {
	MySQLDSN      string
	RedisAddr     string
	HTTPAddr      string
	SessionSecret string
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
	return cfg, nil
}
