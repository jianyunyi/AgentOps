package agentops

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "agentscope-go-sdk/0.1"

type Config struct {
	BaseURL       string
	APIKey        string
	SigningSecret string
	HTTPClient    *http.Client
	Retry         RetryPolicy
	UserAgent     string
}

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type Client struct {
	baseURL       *url.URL
	apiKey        string
	signingSecret []byte
	httpClient    *http.Client
	retry         RetryPolicy
	userAgent     string
}

func NewClient(config Config) (*Client, error) {
	baseURL, err := parseBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("agent api key is required")
	}
	apiKey := strings.TrimSpace(config.APIKey)
	secret, err := decodeSigningSecret(config.SigningSecret)
	if err != nil {
		return nil, errors.New("agent signing secret is invalid")
	}
	retry := config.Retry
	if retry.MaxAttempts == 0 {
		retry.MaxAttempts = 3
	}
	if retry.MaxAttempts < 1 || retry.MaxAttempts > 5 {
		return nil, errors.New("retry max attempts must be between 1 and 5")
	}
	if retry.BaseDelay <= 0 {
		retry.BaseDelay = 100 * time.Millisecond
	}
	if retry.MaxDelay <= 0 {
		retry.MaxDelay = 10 * time.Second
	}
	if retry.MaxDelay < retry.BaseDelay {
		return nil, errors.New("retry max delay must not be less than base delay")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	userAgent := strings.TrimSpace(config.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Client{baseURL: baseURL, apiKey: apiKey, signingSecret: secret, httpClient: httpClient, retry: retry, userAgent: userAgent}, nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("base URL must use http or https")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed, nil
}

func decodeSigningSecret(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("signing secret is required")
	}
	secret, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil || len(secret) != 32 {
		secret, err = base64.StdEncoding.DecodeString(raw)
	}
	if err != nil || len(secret) != 32 {
		return nil, errors.New("signing secret must be a base64-encoded 32-byte value")
	}
	return append([]byte(nil), secret...), nil
}
