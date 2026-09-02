package trace

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
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

func (f *fakeTraceRepository) ListTracesScoped(_ context.Context, tenantID, agentID string, offset, limit int) ([]Trace, int64, error) {
	var result []Trace
	for _, trace := range f.traces {
		if trace.TenantID == tenantID && (agentID == "" || trace.AgentID == agentID) {
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

func (f *fakeTraceRepository) FindTraceScoped(_ context.Context, tenantID, agentID, traceID string) (*Trace, error) {
	for _, trace := range f.traces {
		if trace.TenantID == tenantID && trace.TraceID == traceID && (agentID == "" || trace.AgentID == agentID) {
			return trace, nil
		}
	}
	return nil, ErrTraceNotFound
}

func (f *fakeTraceRepository) ListSpansScoped(_ context.Context, tenantID, agentID, traceID string) ([]Span, error) {
	trace, err := f.FindTraceScoped(context.Background(), tenantID, agentID, traceID)
	if err != nil {
		return nil, err
	}
	var result []Span
	for _, span := range f.spans {
		if span.TenantID == trace.TenantID && span.TraceID == traceID {
			result = append(result, *span)
		}
	}
	return result, nil
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

func (f *fakeAuthenticator) AuthenticateIngestRequest(ctx context.Context, key string, _ agent.AuthenticationMetadata) (agent.Identity, error) {
	return f.AuthenticateAPIKey(ctx, key)
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
	req.Header.Set("X-Agent-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("X-Agent-Nonce", "nonce-tenant-one")
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

func TestTraceAgentKeyQueriesOnlyItsOwnAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeTraceRepository{
		eventIDs: map[string]bool{},
		traces: []*Trace{
			{TenantID: "tenant_001", AgentID: "agent_001", TraceID: "trace_001"},
			{TenantID: "tenant_001", AgentID: "agent_002", TraceID: "trace_002"},
		},
	}
	auth := &fakeAuthenticator{identities: map[string]agent.Identity{
		"ag_live_agent_one": {TenantID: "tenant_001", AgentID: "agent_001"},
	}}
	router := NewRouter(NewService(repo), repo, auth)

	listRequest := httptest.NewRequest("GET", "/api/v1/traces", nil)
	listRequest.Header.Set("Authorization", "Bearer ag_live_agent_one")
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != 200 {
		t.Fatalf("agent trace list status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}
	var page struct {
		Data       []Trace `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Pagination.Total != 1 || len(page.Data) != 1 || page.Data[0].TraceID != "trace_001" {
		t.Fatalf("agent trace list = %+v, want only agent_001 trace", page)
	}

	getRequest := httptest.NewRequest("GET", "/api/v1/traces/trace_002", nil)
	getRequest.Header.Set("Authorization", "Bearer ag_live_agent_one")
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != 404 {
		t.Fatalf("other-agent trace lookup status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
}
