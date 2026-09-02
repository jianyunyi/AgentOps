# P1-5 Agent Go SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a standalone, zero-runtime-dependency Go SDK that signs and reliably ingests AgentScope events.

**Architecture:** Create an independent Go module under `sdk/go` so external applications can import it without depending on server `internal` packages. Keep protocol signing, HTTP transport/retry, public event types, and error parsing in focused files; serialize the body once and regenerate only request authentication headers on retries.

**Tech Stack:** Go standard library (`net/http`, `crypto/hmac`, `crypto/sha256`, `crypto/rand`, `encoding/json`, `httptest`), Go modules, existing AgentScope HMAC v1 protocol.

---

### Task 1: Create the SDK module and public event/config types

**Files:**
- Create: `sdk/go/go.mod`
- Create: `sdk/go/agentops/event.go`
- Create: `sdk/go/agentops/config.go`
- Test: `sdk/go/agentops/config_test.go`

- [ ] **Step 1: Write failing configuration tests**

Test that an empty URL, non-HTTP(S) URL, empty API key, empty signing secret, and retry count outside 1–5 return errors; test that a valid config receives the default 10-second timeout and 3 attempts.

- [ ] **Step 2: Run the focused test**

Run `go test ./sdk/go/agentops -run TestNewClientConfig -count=1` from the repository root. Expected: FAIL because the SDK package does not exist.

- [ ] **Step 3: Implement module and types**

Define module `github.com/jianyunyi/AgentOps/sdk/go`, public `Event`, `IngestResult`, `Config`, `RetryPolicy`, `Client`, and `NewClient`. Validate absolute `http`/`https` URLs, trim one trailing slash, require both credentials, default an injected `http.Client` with a 10-second timeout, and default retries to 3.

- [ ] **Step 4: Run the focused test**

Run `go test ./sdk/go/agentops -run TestNewClientConfig -count=1`. Expected: PASS.

### Task 2: Implement protocol signing and nonce generation

**Files:**
- Create: `sdk/go/agentops/signing.go`
- Test: `sdk/go/agentops/signing_test.go`

- [ ] **Step 1: Write failing protocol-vector tests**

Assert the exact body hash, canonical string, lowercase `v1=` signature, and that two generated Nonces are distinct printable ASCII values of the documented length.

- [ ] **Step 2: Run the focused test**

Run `go test ./sdk/go/agentops -run 'Test(Signature|Nonce)' -count=1`. Expected: FAIL because signing functions do not exist.

- [ ] **Step 3: Implement signing**

Implement `hashBody`, `canonicalRequest`, `signRequest`, and `newNonce` with standard library primitives. Use 32 random bytes encoded as lowercase hex for each Nonce and never expose the signing secret from this file.

- [ ] **Step 4: Run the focused test**

Run the same command and expect PASS.

### Task 3: Implement typed errors and safe response parsing

**Files:**
- Create: `sdk/go/agentops/errors.go`
- Test: `sdk/go/agentops/errors_test.go`

- [ ] **Step 1: Write failing error tests**

Test JSON error parsing for status/code/message, `X-Request-ID`, and `Retry-After`; assert that `Error()` never contains a supplied Authorization value or request body. Test retry classification for transport errors and 408/429/5xx versus 401/409/other 4xx.

- [ ] **Step 2: Implement safe errors**

Define `APIError` with `StatusCode`, `Code`, `Message`, `RequestID`, and `RetryAfter`; parse only the documented JSON fields and cap message length. Add `isRetryableStatus` and `retryAfter` helpers.

- [ ] **Step 3: Run tests**

Run `go test ./sdk/go/agentops -run 'Test(APIError|Retry)' -count=1`. Expected: PASS.

### Task 4: Implement ingest transport and retry loop

**Files:**
- Create: `sdk/go/agentops/client.go`
- Create: `sdk/go/agentops/retry.go`
- Test: `sdk/go/agentops/client_test.go`

- [ ] **Step 1: Write failing HTTP tests**

Use `httptest.Server` to assert `POST /api/v1/ingest/events`, exact JSON bytes, auth headers, correct signature verification, `202` duplicate parsing, and fresh Nonces across a first-503/second-202 retry. Add tests proving 401 and 409 are not retried and Context cancellation interrupts backoff.

- [ ] **Step 2: Implement one request attempt**

Marshal the event once, create a Context-bound request with a fresh Timestamp/Nonce/signature, set `Content-Type`, `Authorization`, `X-Agent-Timestamp`, `X-Agent-Nonce`, `X-Agent-Signature`, and a generated `X-Request-ID`, then close the response body on every path.

- [ ] **Step 3: Implement retry loop**

Retry only the classified transport/status failures while attempts remain. Use `Retry-After` capped at 10 seconds for 429 and Context-aware timers for backoff. Reuse the serialized body and event ID, but create new auth headers for every attempt.

- [ ] **Step 4: Run tests**

Run `go test ./sdk/go/agentops -count=1`. Expected: PASS.

### Task 5: Add integration test, examples, and documentation

**Files:**
- Create: `sdk/go/agentops/integration_test.go`
- Create: `sdk/go/examples/ingest/main.go`
- Create: `sdk/go/README.md`
- Modify: `docs/superpowers/specs/2026-09-02-p1-5-agent-go-sdk-design.md` only if review finds an inconsistency

- [ ] **Step 1: Add opt-in real integration test**

When `AGENTSCOPE_INTEGRATION=1`, construct the SDK with the service endpoint and credentials supplied by environment variables, send a unique valid event, and verify HTTP 202. Skip without the flag so normal CI has no external dependency.

- [ ] **Step 2: Add safe example and README**

Document `go get github.com/jianyunyi/AgentOps/sdk/go`, environment-based credential loading, minimal ingest code, retry/idempotency semantics, HMAC header protocol, and the requirement to rotate legacy Agents before enabling required signatures.

- [ ] **Step 3: Run SDK tests and vet**

Run `go test ./sdk/go/... -count=1` and `go vet ./sdk/go/...`. Expected: PASS.

### Task 6: Full verification and delivery

**Files:**
- No additional source files.

- [ ] **Step 1: Run repository checks**

Run `go test ./... -count=1`, `go vet ./...`, `go test ./sdk/go/... -count=1`, and the frontend test/build commands. Expected: all PASS.

- [ ] **Step 2: Run real dependency integration**

Use `MYSQL_DSN` pointed at MySQL `127.0.0.1:3307` and `REDIS_ADDR=127.0.0.1:3509`; run the existing integration suite plus the SDK integration test. Expected: all PASS.

- [ ] **Step 3: Inspect secrets and diff**

Run `git diff --check` and search the SDK for `APIKey`, `SigningSecret`, `Authorization`, and request-body logging. Confirm only header construction uses credentials and no example contains a real secret.

- [ ] **Step 4: Commit and push**

Commit as `feat: add agent go sdk` and push `main` after the worktree and tests are clean.
