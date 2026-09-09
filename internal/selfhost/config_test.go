package selfhost

import (
	"strings"
	"testing"
)

func TestParseConfigDerivesLocalOriginAndGeneratesSecrets(t *testing.T) {
	config, err := ParseConfig(strings.NewReader("WEB_PORT=3300\nAPI_PORT=8080\nMYSQL_PASSWORD=local\n"), true)
	if err != nil { t.Fatal(err) }
	if config.Values["WEB_ORIGIN"] != "http://127.0.0.1:3300" { t.Fatalf("origin = %q", config.Values["WEB_ORIGIN"]) }
	if config.Values["SESSION_SECRET"] == "" || config.Values["AGENT_SIGNING_ENCRYPTION_KEY"] == "" { t.Fatal("expected generated secrets") }
}

func TestParseConfigRejectsUnsafeInput(t *testing.T) {
	for _, input := range []string{"UNKNOWN=value\n", "WEB_PORT=8080\nWEB_PORT=8081\n", "WEB_PORT=$HOME\n", "WEB_PORT=70000\n"} {
		if _, err := ParseConfig(strings.NewReader(input), false); err == nil { t.Fatalf("accepted %q", input) }
	}
}
