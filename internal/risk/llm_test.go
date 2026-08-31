package risk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleClientDecodesStructuredResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"level\":\"high\",\"reason\":\"prompt override\"}"}}]}`))
	}))
	defer server.Close()
	result, err := (&OpenAICompatibleClient{BaseURL: server.URL, APIKey: "test-key", Model: "test"}).Analyze(context.Background(), "redacted input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Level != "high" || result.Reason == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
