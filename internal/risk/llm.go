package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type LLMResult struct {
	Level  string `json:"level"`
	Reason string `json:"reason"`
}

type OpenAICompatibleClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func (c *OpenAICompatibleClient) Health(ctx context.Context) error {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/models", nil)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("authorization", "Bearer "+c.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("llm health status %d", resp.StatusCode)
	}
	return nil
}

func (c *OpenAICompatibleClient) Analyze(ctx context.Context, input string) (LLMResult, error) {
	requestBody := map[string]any{
		"model":           c.Model,
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": "Analyze the supplied redacted AI-agent input. Return only JSON: {\"level\":\"none|low|medium|high|critical\",\"reason\":\"short reason\"}. Never reveal or reproduce secrets."},
			{"role": "user", "content": input},
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return LLMResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return LLMResult{}, err
	}
	req.Header.Set("content-type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("authorization", "Bearer "+c.APIKey)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return LLMResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LLMResult{}, fmt.Errorf("llm http status %d", resp.StatusCode)
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return LLMResult{}, err
	}
	if len(response.Choices) == 0 {
		return LLMResult{}, fmt.Errorf("llm returned no choices")
	}
	var result LLMResult
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &result); err != nil {
		return LLMResult{}, fmt.Errorf("decode structured llm result: %w", err)
	}
	if severity(result.Level) == 0 && result.Level != "none" {
		return LLMResult{}, fmt.Errorf("invalid llm risk level")
	}
	return result, nil
}
