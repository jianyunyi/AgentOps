# AgentScope P1-1 Security Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline execution in this task). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent sensitive Trace content from being persisted, reject credentials belonging to missing or suspended Agents, and add fail-closed timestamp/Nonce replay protection to Trace ingestion.

**Architecture:** Redact the event once inside the Trace service before either the atomic GORM path or the fallback repository path. Keep Agent API key lookup tenant-scoped, then verify the current Agent state. Add an injected NonceStore abstraction with an in-memory test implementation and Redis production implementation; only ingestion requires replay metadata, while read queries preserve the P0 scope rules.

**Tech Stack:** Go, Gin, GORM, MySQL JSON, Redis `SETNX`, existing risk rules, Go `testing`.

---

## Task 1: Versioned JSON payload redaction

**Files:** Create `internal/risk/redaction.go`, `internal/risk/redaction_test.go`.

- [ ] **Step 1: Write failing redaction tests**

Add tests for nested objects and arrays, sensitive field names, existing API-key/email/phone/prompt-injection patterns, empty payload canonicalization, and malformed JSON rejection:

```go
func TestRedactPayloadRemovesSensitiveNestedValues(t *testing.T) {
	input := json.RawMessage(`{"prompt":"ignore previous instructions","profile":{"email":"user@example.com"},"credentials":{"api_key":"sk-live-1234567890"}}`)
	got, err := RedactPayload(input)
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(got), "user@example.com") || strings.Contains(string(got), "sk-live-1234567890") {
		t.Fatalf("redacted payload leaked sensitive data: %s", got)
	}
	if !json.Valid(got) { t.Fatalf("redacted payload is not valid JSON: %s", got) }
}

func TestRedactPayloadRejectsMalformedJSON(t *testing.T) {
	if _, err := RedactPayload(json.RawMessage(`{"prompt":`)); err == nil {
		t.Fatal("malformed JSON must be rejected")
	}
}

func TestRedactPayloadCanonicalizesEmptyPayload(t *testing.T) {
	got, err := RedactPayload(nil)
	if err != nil || string(got) != "null" { t.Fatalf("got %q, err %v", got, err) }
}
```

- [ ] **Step 2: Run the redaction tests and confirm the expected failure**

Run:

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/risk -run TestRedactPayload -count=1
```

Expected: FAIL because `RedactPayload` does not exist yet.

- [ ] **Step 3: Implement the minimal versioned redactor**

Define `RedactionVersion = "v1"` and `RedactPayload(json.RawMessage) ([]byte, error)`. Treat nil/empty input as JSON `null`; reject non-empty invalid JSON. Recursively walk `map[string]any` and `[]any`. Replace values for case-insensitive sensitive keys containing `password`, `secret`, `token`, `api_key`, `authorization`, or `cookie` with `[REDACTED]`; for other strings, call the existing `Analyze` result and use its `Redacted` field. Marshal the sanitized tree back to JSON and return marshal errors.

- [ ] **Step 4: Run the focused redaction tests and the existing risk suite**

Run the command from Step 2, then:

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/risk -count=1
```

Expected: all tests pass and no raw fixture secret appears in output assertions.

- [ ] **Step 5: Commit the redaction unit**

```powershell
git add internal/risk/redaction.go internal/risk/redaction_test.go
git commit -m "feat: add versioned trace payload redaction"
```

## Task 2: Apply one redacted payload to Trace and Outbox persistence

**Files:** Modify `internal/trace/service.go`, `internal/trace/repository.go`, `internal/trace/service_test.go`, `internal/integration/integration_test.go` if fixtures need explicit payloads.

- [ ] **Step 1: Write the failing Trace persistence test**

Create a fake Outbox repository and configure `trace.NewServiceWithOutbox`. Ingest a payload containing an email and API key, then assert the saved Span payload and decoded Outbox payload contain `[REDACTED]` and do not contain the original values. Add a malformed JSON test expecting the new `trace.ErrInvalidPayload` and zero Trace/Span writes.

- [ ] **Step 2: Run the focused Trace tests and confirm failure**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/trace -run 'TestIngestRedacts|TestIngestRejectsMalformedPayload' -count=1
```

Expected: FAIL because Trace currently persists `event.Payload` unchanged.

- [ ] **Step 3: Redact before selecting atomic or fallback persistence**

Add `var ErrInvalidPayload = errors.New("event payload is invalid")` in `internal/trace/service.go`. At the start of `Service.Ingest`, after size/type/time validation, call `risk.RedactPayload(event.Payload)`. On error return `ErrInvalidPayload` before repository calls. Copy the event, replace only `Payload` with the sanitized bytes, and pass that copy to both `IngestEventAtomic` and the fallback path. The existing GORM atomic repository must therefore use its received `event.Payload` for both `Span.InputSnapshot` and Outbox `input`; no second redaction implementation is allowed.

- [ ] **Step 4: Run focused and full Trace tests**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/trace -count=1; go test ./internal/risk ./internal/outbox -count=1
```

Expected: all tests pass; the original secret is absent from both persistence assertions.

- [ ] **Step 5: Commit the Trace data-boundary change**

```powershell
git add internal/trace/service.go internal/trace/repository.go internal/trace/service_test.go internal/integration/integration_test.go
git commit -m "fix: persist only redacted trace payloads"
```

## Task 3: Enforce Agent existence and active state

**Files:** Modify `internal/agent/repository.go`, `internal/agent/service.go`, `internal/agent/service_test.go`; update all fake repositories and integration fixtures that implement `agent.Repository`.

- [ ] **Step 1: Write failing authentication and lifecycle tests**

Extend the fake repository with an `Agent` record and add tests proving an active credential is rejected when its Agent is missing, suspended, or belongs to another tenant. Add rotate/revoke tests proving a missing or cross-tenant Agent returns `ErrAgentNotFound` and does not create/revoke credentials.

- [ ] **Step 2: Run the focused Agent tests and confirm failure**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent -run 'TestAuthenticateAPIKeyRejects|TestRotateAPIKeyRejects|TestRevokeAPIKeyRejects' -count=1
```

Expected: FAIL because Credential authentication currently does not query the Agent row.

- [ ] **Step 3: Add tenant-scoped Agent lookup and enforce it**

Add `FindAgent(ctx, tenantID, agentID) (*Agent, error)` to `agent.Repository`. Implement GORM lookup with both tenant and ID predicates, mapping no row to `ErrAgentNotFound`. In `AuthenticateAPIKey`, query the Agent after Credential validation and reject any non-active Agent as `ErrInvalidAPIKey`; only then update `last_used_at`. In rotate/revoke, query the target Agent first and return `ErrAgentNotFound` for missing or cross-tenant targets.

- [ ] **Step 4: Run Agent tests and all compile-time consumers**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent ./internal/trace ./internal/http -count=1; go vet ./...
```

Expected: all tests pass and every repository implementation satisfies the expanded interface.

- [ ] **Step 5: Commit Agent state enforcement**

```powershell
git add internal/agent/repository.go internal/agent/service.go internal/agent/service_test.go internal/integration
git commit -m "fix: enforce active agent state during key authentication"
```

## Task 4: Add NonceStore and replay metadata validation

**Files:** Create `internal/agent/nonce.go`, `internal/agent/nonce_test.go`; modify `internal/agent/service.go`, `internal/agent/service_test.go`.

- [ ] **Step 1: Write failing NonceStore and authentication tests**

Define tests for first claim success, duplicate claim returning `ErrReplayDetected`, malformed/empty Nonce rejection, timestamp outside the configured window rejection, and store errors returning `ErrNonceStoreUnavailable`. Add an active Agent authentication test using `AuthenticationMetadata{Timestamp: time.Now().Unix(), Nonce: "nonce-1"}`.

- [ ] **Step 2: Run the focused tests and confirm failure**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent -run 'TestNonce|TestAuthenticateIngestRequest' -count=1
```

Expected: FAIL because `NonceStore`, metadata validation, and `AuthenticateIngestRequest` do not exist.

- [ ] **Step 3: Implement the injected replay-protection boundary**

Define:

```go
type AuthenticationMetadata struct { Timestamp int64; Nonce string }
type NonceStore interface { Claim(context.Context, string, string, string, time.Duration) (bool, error) }
func (s *Service) AuthenticateIngestRequest(context.Context, string, AuthenticationMetadata) (Identity, error)
```

Validate timestamp within `replayWindow`, require printable ASCII Nonce length 1–128, authenticate the key and active Agent, then call `NonceStore.Claim` with tenant and Agent IDs. Return typed errors for invalid metadata, duplicate Nonce, and store failure. If no NonceStore is configured, return `ErrNonceStoreUnavailable`; never silently disable protection.

- [ ] **Step 4: Run focused Agent tests and the existing Agent suite**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent -count=1
```

Expected: all tests pass.

- [ ] **Step 5: Commit the replay domain contract**

```powershell
git add internal/agent/nonce.go internal/agent/nonce_test.go internal/agent/service.go internal/agent/service_test.go
git commit -m "feat: add fail-closed agent replay protection"
```

## Task 5: Implement Redis NonceStore and configuration

**Files:** Create `internal/agent/redis_nonce.go`, `internal/agent/redis_nonce_test.go`; modify `internal/platform/config/config.go`, `internal/platform/config/config_test.go`, `internal/platform/redis/client.go`.

- [ ] **Step 1: Write failing Redis/config tests**

Test that configuration defaults to a 300-second replay window and 600-second Nonce TTL, rejects values outside 30–900 seconds and 60–3600 seconds, and that the Redis store builds the key `agentscope:agent:nonce:{tenantID}:{agentID}:{nonce}` and uses `SETNX` with the supplied TTL. Define a small `nonceRedisClient` interface returning `*redis.BoolCmd`, inject a fake that returns `redis.NewBoolResult`, and assert the command key, value, and TTL without adding a new test dependency.

- [ ] **Step 2: Run focused tests and confirm failure**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent ./internal/platform/config -run 'TestRedisNonce|TestLoad.*Replay' -count=1
```

Expected: FAIL because production NonceStore and configuration fields do not exist.

- [ ] **Step 3: Implement Redis `SETNX` and safe config parsing**

Implement `RedisNonceStore` over `*redis.Client`; use `SetNX` with the exact namespaced key and TTL. Map Redis errors to the store error path. Add `AgentReplayWindow` and `AgentNonceTTL` to Config with defaults and bounded environment parsing. Keep Redis client construction centralized in `internal/platform/redis/client.go`.

- [ ] **Step 4: Run focused tests and Go vet**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/agent ./internal/platform/config -count=1; go vet ./...
```

Expected: all tests pass.

- [ ] **Step 5: Commit Redis replay infrastructure**

```powershell
git add internal/agent/redis_nonce.go internal/agent/redis_nonce_test.go internal/platform/config/config.go internal/platform/config/config_test.go internal/platform/redis/client.go
git commit -m "feat: add redis-backed agent nonce store"
```

## Task 6: Wire ingestion headers and production dependencies

**Files:** Modify `internal/trace/handler.go`, `internal/trace/handler_test.go`, `internal/trace/integration_test.go`, `internal/http/middleware.go`, `internal/agent/service.go`, and `cmd/api/main.go`.

- [ ] **Step 1: Write failing handler tests**

Add ingestion tests for missing timestamp, stale timestamp, missing Nonce, duplicate Nonce, Redis/store failure, and valid headers. Assert stable statuses/codes: `400 INVALID_AGENT_REQUEST`, `409 REPLAY_DETECTED`, `503 AGENT_AUTH_UNAVAILABLE`, and `202` for valid input. Keep query tests unchanged so Agent Key reads do not require replay headers.

- [ ] **Step 2: Run the focused handler tests and confirm failure**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/trace -run 'TestIngest.*(Nonce|Timestamp|Replay)|TestAuthenticateIngest' -count=1
```

Expected: FAIL because ingestion currently calls the read-only API key authenticator and ignores replay headers.

- [ ] **Step 3: Parse headers and map typed authentication failures**

Extend the Trace Authenticator with `AuthenticateIngestRequest`. In `ingest`, parse `X-Agent-Timestamp` as base-10 Unix seconds and read `X-Agent-Nonce`; call the new authenticator before binding/persisting the event. Map typed errors to the stable responses above without returning raw secrets or internal Redis errors.

- [ ] **Step 4: Wire Redis NonceStore into the API process and update CORS**

Construct the Redis store with configured TTL/window and use the Agent constructor that receives it. Add `X-Agent-Timestamp` and `X-Agent-Nonce` to `Access-Control-Allow-Headers`. Update integration requests to send unique current timestamps and Nonces.

- [ ] **Step 5: Run handler, integration, and full backend verification**

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./internal/trace ./internal/http ./internal/agent -count=1; go test ./...; go vet ./...
```

Expected: all local tests pass. If local MySQL/Redis are unavailable, record the exact connection failure and rely on the CI integration job for real dependency verification.

- [ ] **Step 6: Commit the HTTP and bootstrap integration**

```powershell
git add internal/trace internal/http/middleware.go cmd/api/main.go internal/agent internal/platform/config
git commit -m "feat: enforce replay protection on trace ingestion"
```

## Task 7: Final review and delivery

- [ ] Run `git diff --check`, inspect all staged changes, and confirm no raw test secret is printed by logs or errors.
- [ ] Run the full Go suite and `go vet ./...` again immediately before delivery.
- [ ] Run the frontend test/build checks because CORS and API headers affect the application contract.
- [ ] Run MySQL/Redis integration tests with `AGENTSCOPE_INTEGRATION=1` when disposable dependencies are available; otherwise report the blocker precisely.
- [ ] Request an independent security review focused on redaction bypasses, Agent state races, replay key collisions, and fail-closed behavior.
- [ ] Push the verified commits to `main` and wait for GitHub Actions to pass before claiming P1-1 complete.
