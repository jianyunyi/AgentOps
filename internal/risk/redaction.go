package risk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const RedactionVersion = "v1"

const redactionMarker = "[REDACTED]"

// RedactPayload validates and recursively redacts an event payload while keeping it valid JSON.
func RedactPayload(payload json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return []byte("null"), nil
	}

	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	redacted := redactValue(value, "")
	result, err := json.Marshal(redacted)
	if err != nil {
		return nil, fmt.Errorf("encode redacted payload: %w", err)
	}
	return result, nil
}

func redactValue(value any, key string) any {
	if isSensitiveKey(key) {
		return redactionMarker
	}

	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			typed[childKey] = redactValue(childValue, childKey)
		}
		return typed
	case []any:
		for index, childValue := range typed {
			typed[index] = redactValue(childValue, "")
		}
		return typed
	case string:
		return Analyze(typed).Redacted
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	var normalized strings.Builder
	for _, character := range strings.ToLower(key) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			normalized.WriteRune(character)
		}
	}
	name := normalized.String()
	for _, marker := range []string{"password", "secret", "token", "apikey", "authorization", "cookie"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}
