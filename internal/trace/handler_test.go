package trace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"agentscope/internal/agent"
	"agentscope/internal/auth"
	"github.com/gin-gonic/gin"
)

type metadataCapturingAuthenticator struct {
	identity agent.Identity
	metadata agent.AuthenticationMetadata
}

func (f *metadataCapturingAuthenticator) AuthenticateAPIKey(context.Context, string) (agent.Identity, error) {
	return f.identity, nil
}

func (f *metadataCapturingAuthenticator) AuthenticateIngestRequest(_ context.Context, _ string, metadata agent.AuthenticationMetadata) (agent.Identity, error) {
	f.metadata = metadata
	return f.identity, nil
}

type replayTestAuthenticator struct {
	identity   agent.Identity
	ingestErr  error
	ingestCall int
}

func (f *replayTestAuthenticator) AuthenticateAPIKey(context.Context, string) (agent.Identity, error) {
	return f.identity, nil
}

func (f *replayTestAuthenticator) AuthenticateIngestRequest(_ context.Context, _ string, _ agent.AuthenticationMetadata) (agent.Identity, error) {
	f.ingestCall++
	if f.ingestErr != nil {
		return agent.Identity{}, f.ingestErr
	}
	return f.identity, nil
}

func TestAuthenticateQueryRequiresAgentReadPermissionForSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("tenant_id", "tenant_001")
	context.Set("user_role", auth.RoleAuditor)

	identity, ok := (&Handler{}).authenticateQuery(context)
	if ok || identity.TenantID != "" {
		t.Fatal("session without agent:read must not authenticate a trace query")
	}
	if context.Writer.Status() != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", context.Writer.Status(), http.StatusForbidden)
	}
}

func TestAuthenticateQueryRejectsMixedSessionAndAgentCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("tenant_id", "tenant_001")
	context.Set("user_role", auth.RoleOwner)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil)
	context.Request.Header.Set("Authorization", "Bearer agent-key")

	identity, ok := (&Handler{}).authenticateQuery(context)
	if ok || identity.TenantID != "" || identity.AgentID != "" {
		t.Fatal("mixed session and agent credentials must not authenticate a trace query")
	}
	if context.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", context.Writer.Status(), http.StatusBadRequest)
	}
}

func TestIngestRequiresReplayHeaders(t *testing.T) {
	authenticator := &replayTestAuthenticator{identity: agent.Identity{TenantID: "tenant_001", AgentID: "agent_001"}}
	router := NewRouter(NewService(&fakeTraceRepository{eventIDs: map[string]bool{}}), &fakeTraceRepository{eventIDs: map[string]bool{}}, authenticator)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(`{"event_id":"evt_1","trace_id":"trace_1","span_id":"span_1","event_type":"trace_start","occurred_at":"2026-09-02T00:00:00Z","payload":null}`))
	request.Header.Set("Authorization", "Bearer ag_live_test")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "INVALID_AGENT_REQUEST") {
		t.Fatalf("missing replay headers response = %d %s", response.Code, response.Body.String())
	}
	if authenticator.ingestCall != 0 {
		t.Fatal("missing replay headers must be rejected before ingest authentication")
	}
}

func TestIngestUsesReplayAwareAuthentication(t *testing.T) {
	authenticator := &replayTestAuthenticator{identity: agent.Identity{TenantID: "tenant_001", AgentID: "agent_001"}}
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	router := NewRouter(NewService(repo), repo, authenticator)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(`{"event_id":"evt_2","trace_id":"trace_2","span_id":"span_2","event_type":"trace_start","occurred_at":"2026-09-02T00:00:00Z","payload":null}`))
	request.Header.Set("Authorization", "Bearer ag_live_test")
	request.Header.Set("X-Agent-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	request.Header.Set("X-Agent-Nonce", "nonce-2")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("valid replay headers response = %d %s", response.Code, response.Body.String())
	}
	if authenticator.ingestCall != 1 {
		t.Fatalf("ingest authentication calls = %d, want 1", authenticator.ingestCall)
	}
}

func TestIngestMapsReplayDetectedError(t *testing.T) {
	authenticator := &replayTestAuthenticator{identity: agent.Identity{TenantID: "tenant_001", AgentID: "agent_001"}, ingestErr: agent.ErrReplayDetected}
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	router := NewRouter(NewService(repo), repo, authenticator)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(`{"event_id":"evt_3","trace_id":"trace_3","span_id":"span_3","event_type":"trace_start","occurred_at":"2026-09-02T00:00:00Z","payload":null}`))
	request.Header.Set("Authorization", "Bearer ag_live_test")
	request.Header.Set("X-Agent-Timestamp", "1756771200")
	request.Header.Set("X-Agent-Nonce", "nonce-3")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "REPLAY_DETECTED") {
		t.Fatalf("replay response = %d %s", response.Code, response.Body.String())
	}
}

func TestIngestPassesBodyBoundSignatureMetadataToAuthenticator(t *testing.T) {
	authenticator := &metadataCapturingAuthenticator{identity: agent.Identity{TenantID: "tenant_001", AgentID: "agent_001"}}
	repo := &fakeTraceRepository{eventIDs: map[string]bool{}}
	router := NewRouter(NewService(repo), repo, authenticator)
	body := `{"event_id":"evt_4","trace_id":"trace_4","span_id":"span_4","event_type":"trace_start","occurred_at":"2026-09-02T00:00:00Z","payload":null}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/events", strings.NewReader(body))
	timestamp := time.Now().Unix()
	nonce := "nonce-4"
	request.Header.Set("Authorization", "Bearer ag_live_test")
	request.Header.Set("X-Agent-Timestamp", strconv.FormatInt(timestamp, 10))
	request.Header.Set("X-Agent-Nonce", nonce)
	request.Header.Set("X-Agent-Signature", "v1=test-signature")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("signed metadata response = %d %s", response.Code, response.Body.String())
	}
	hash := sha256.Sum256([]byte(body))
	if authenticator.metadata.Method != http.MethodPost || authenticator.metadata.Path != "/api/v1/ingest/events" || authenticator.metadata.BodyHash != hex.EncodeToString(hash[:]) || authenticator.metadata.Signature != "v1=test-signature" {
		t.Fatalf("captured metadata = %+v", authenticator.metadata)
	}
}
