package trace

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agentscope/internal/agent"
	"github.com/gin-gonic/gin"
)

type fakeAuthenticator struct {
	identities map[string]agent.Identity
}

func (f *fakeTraceRepository) ListTraces(_ context.Context, tenantID string, offset, limit int) ([]Trace, int64, error) {
	var result []Trace
	for _, trace := range f.traces {
		if trace.TenantID == tenantID {
			result = append(result, *trace)
		}
	}
	if offset >= len(result) {
		return []Trace{}, int64(len(result)), nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], int64(len(result)), nil
}

func (f *fakeTraceRepository) ListSpans(_ context.Context, tenantID, traceID string) ([]Span, error) {
	var result []Span
	for _, span := range f.spans {
		if span.TenantID == tenantID && span.TraceID == traceID {
			result = append(result, *span)
		}
	}
	return result, nil
}

func (f *fakeAuthenticator) AuthenticateAPIKey(_ context.Context, key string) (agent.Identity, error) {
	identity, ok := f.identities[key]
	if !ok {
		return agent.Identity{}, agent.ErrInvalidAPIKey
	}
	return identity, nil
}

func TestTraceTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	traceService := NewService(repo)
	auth := &fakeAuthenticator{identities: map[string]agent.Identity{
		"ag_live_tenant_one": {TenantID: "tenant_001", AgentID: "agent_001"},
		"ag_live_tenant_two": {TenantID: "tenant_002", AgentID: "agent_002"},
	}}
	router := NewRouter(traceService, repo, auth)

	payload := Event{
		EventID:    "evt_001",
		TraceID:    "trace_001",
		SpanID:     "span_001",
		EventType:  EventTraceStart,
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(`{"input":"check order errors"}`),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/ingest/events", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer ag_live_tenant_one")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 202 {
		t.Fatalf("ingest status = %d, body = %s", res.Code, res.Body.String())
	}

	otherTenantReq := httptest.NewRequest("GET", "/api/v1/traces/trace_001", nil)
	otherTenantReq.Header.Set("Authorization", "Bearer ag_live_tenant_two")
	otherTenantRes := httptest.NewRecorder()
	router.ServeHTTP(otherTenantRes, otherTenantReq)
	if otherTenantRes.Code != 404 {
		t.Fatalf("cross-tenant trace lookup status = %d, body = %s", otherTenantRes.Code, otherTenantRes.Body.String())
	}
}
