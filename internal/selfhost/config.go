package selfhost

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type Config struct{ Values map[string]string }

var allowed = map[string]bool{"AGENTOPS_WEB_IMAGE": true, "AGENTOPS_API_IMAGE": true, "WEB_PORT": true, "API_PORT": true, "MYSQL_PASSWORD": true, "SESSION_SECRET": true, "AGENT_SIGNING_ENCRYPTION_KEY": true, "ENABLE_LOCAL_LLM": true, "LLM_MODEL": true}

func ParseConfig(r io.Reader, generate bool) (Config, error) {
	values := map[string]string{}
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || !allowed[parts[0]] || hasKey(values, parts[0]) || strings.ContainsAny(parts[1], "\r\n$") {
			return Config{}, fmt.Errorf("invalid configuration")
		}
		values[parts[0]] = parts[1]
	}
	if err := s.Err(); err != nil {
		return Config{}, err
	}
	for _, key := range []string{"WEB_PORT", "API_PORT"} {
		if value, ok := values[key]; ok {
			port, err := strconv.Atoi(value)
			if err != nil || port < 1 || port > 65535 {
				return Config{}, fmt.Errorf("invalid %s", key)
			}
		}
	}
	if values["WEB_PORT"] == "" {
		values["WEB_PORT"] = "3300"
	}
	if values["API_PORT"] == "" {
		values["API_PORT"] = "8080"
	}
	if values["WEB_PORT"] == values["API_PORT"] {
		return Config{}, fmt.Errorf("ports must differ")
	}
	values["WEB_ORIGIN"] = "http://127.0.0.1:" + values["WEB_PORT"]
	if generate {
		for _, key := range []string{"SESSION_SECRET", "AGENT_SIGNING_ENCRYPTION_KEY"} {
			if values[key] == "" {
				b := make([]byte, 32)
				if _, err := rand.Read(b); err != nil {
					return Config{}, err
				}
				values[key] = base64.StdEncoding.EncodeToString(b)
			}
		}
	}
	return Config{Values: values}, nil
}

func IsLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	return err == nil && (host == "127.0.0.1" || host == "localhost")
}

func hasKey(values map[string]string, key string) bool { _, ok := values[key]; return ok }
func (c Config) Validate() error {
	for _, key := range []string{"AGENTOPS_API_IMAGE", "AGENTOPS_WEB_IMAGE"} {
		if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./:_-]*@sha256:[a-f0-9]{64}$`).MatchString(c.Values[key]) {
			return fmt.Errorf("%s requires an immutable image digest", key)
		}
	}
	if c.Values["MYSQL_PASSWORD"] == "" || strings.ContainsAny(c.Values["MYSQL_PASSWORD"], "\\\"' #\t") {
		return fmt.Errorf("MYSQL_PASSWORD is missing or contains unsupported env characters")
	}
	for _, key := range []string{"SESSION_SECRET", "AGENT_SIGNING_ENCRYPTION_KEY"} {
		b, err := base64.StdEncoding.DecodeString(c.Values[key])
		if err != nil || len(b) != 32 {
			return fmt.Errorf("%s must be 32 bytes in padded Base64", key)
		}
	}
	return nil
}
