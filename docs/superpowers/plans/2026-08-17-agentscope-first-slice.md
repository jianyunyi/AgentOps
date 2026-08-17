# AgentScope First Vertical Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable AgentScope enterprise slice: create an Agent credential, ingest idempotent Agent events, persist Trace/Span data with tenant isolation, enqueue asynchronous analysis, and display Trace summaries and details in a Next.js console.

**Architecture:** Use a modular Go monolith with Gin, GORM, MySQL, Redis Streams, and a worker process. Keep HTTP handlers thin; application services orchestrate use cases; repositories own persistence; domain services own validation and state rules. Use Next.js App Router with typed API clients for the console.

**Tech Stack:** Go, Gin, GORM, MySQL, Redis Streams, Next.js, React, TypeScript, Vitest/Testing Library, Docker Compose.

---

## Scope and acceptance criteria

This plan covers the first vertical slice only. It does not implement SSO, SDKs, live tool replay, automatic blocking, or production deployment orchestration.

The slice is accepted when:

- An authenticated tenant admin can create an Agent and see its raw API key exactly once.
- The raw key is never persisted; only a hash is stored.
- An Agent can post `trace_start`, `llm_call`, `tool_call`, and `trace_end` events.
- The server derives `tenant_id` and `agent_id` from the API key rather than trusting the payload.
- Duplicate `(tenant_id, event_id)` submissions are idempotent.
- Trace and Span records can be queried only inside the current tenant.
- Events are published to a Redis Stream and a worker can consume and retry them.
- The console lists traces and displays a trace's span tree.
- Unit, integration, and frontend tests cover the critical path.

## File map

### Repository and runtime

- Create `go.mod`: Go module metadata.
- Create `cmd/api/main.go`: API process bootstrap.
- Create `cmd/worker/main.go`: worker process bootstrap.
- Create `internal/platform/config/config.go`: environment configuration.
- Create `internal/platform/database/mysql.go`: GORM connection and migration entry point.
- Create `internal/platform/redis/client.go`: Redis client and stream constants.
- Create `docker-compose.yml`: local MySQL and Redis dependencies.
- Create `.env.example`: documented local configuration.

### Backend domains

- Create `internal/agent/model.go`, `repository.go`, `service.go`, `handler.go`: Agent lifecycle, credential storage, and HTTP endpoints.
- Create `internal/trace/model.go`, `repository.go`, `service.go`, `handler.go`: event ingestion, Trace/Span persistence, and query endpoints.
- Create `internal/worker/stream.go`, `analysis.go`: Redis Stream publishing and idempotent analysis worker.
- Create `internal/http/middleware/auth.go`, `internal/http/router.go`: authentication context and route registration.

### Tests and frontend

- Create `internal/agent/service_test.go`: key creation and authentication behavior.
- Create `internal/trace/service_test.go`, `integration_test.go`: validation, idempotency, and tenant isolation.
- Create `internal/worker/analysis_test.go`: retry and duplicate-consumption behavior.
- Create `web/package.json`, `web/tsconfig.json`: Next.js and test setup.
- Create `web/app/layout.tsx`, `web/app/dashboard/traces/page.tsx`, `web/app/dashboard/traces/[traceId]/page.tsx`.
- Create `web/components/traces/trace-table.tsx`, `span-tree.tsx`.
- Create `web/lib/api/types.ts`, `client.ts`.
- Create `web/components/traces/trace-table.test.tsx`.

## Task 1: Bootstrap the repository and local dependencies

**Files:** `go.mod`, `docker-compose.yml`, `.env.example`, `cmd/api/main.go`, `cmd/worker/main.go`, `internal/platform/config/config.go`, `internal/platform/database/mysql.go`, `internal/platform/redis/client.go`

- [ ] Write a smoke test that loads required configuration and rejects a missing `MYSQL_DSN`.
- [ ] Run `go test ./internal/platform/config -run TestLoad -count=1` and confirm it fails because the package does not exist.
- [ ] Implement strict configuration loading for `MYSQL_DSN`, `REDIS_ADDR`, `HTTP_ADDR`, and `SESSION_SECRET`.
- [ ] Add dependencies for Gin, GORM MySQL, Redis, password hashing, IDs, and test helpers.
- [ ] Add Docker Compose services for MySQL 8 and Redis 7 with health checks and named volumes.
- [ ] Add API and worker entrypoints that construct dependencies without business routes.
- [ ] Run `go test ./...` and `go vet ./...`; both must exit 0.

## Task 2: Implement Agent creation and API-key authentication

**Files:** `internal/agent/model.go`, `repository.go`, `service.go`, `handler.go`, `internal/http/middleware/auth.go`, `internal/agent/service_test.go`

- [ ] Write a failing test proving `CreateAgent` returns a raw key once and persists only a hash plus a display prefix.
- [ ] Run `go test ./internal/agent -run TestCreateAgent -count=1` and confirm it fails before implementation.
- [ ] Define Agent and AgentCredential models with tenant-scoped unique constraints.
- [ ] Implement cryptographically random API-key generation, constant-time verification, revocation, and one-time raw-key return.
- [ ] Add `POST /api/v1/agents`, `GET /api/v1/agents`, and `POST /api/v1/agents/:id/rotate-key`.
- [ ] Add Agent API-key middleware that resolves `tenant_id` and `agent_id`; ignore client-supplied tenant identity.
- [ ] Run focused Agent tests and `go test ./...`.

## Task 3: Implement Trace/Span persistence and event validation

**Files:** `internal/trace/model.go`, `repository.go`, `service.go`, `service_test.go`

- [ ] Write failing tests for valid acceptance, invalid event type, future timestamp, maximum payload, and duplicate idempotency.
- [ ] Run `go test ./internal/trace -run TestIngest -count=1` and confirm the tests fail for missing service behavior.
- [ ] Define event, Trace, Span, LLMCall, and ToolCall models with `(tenant_id, event_id)` uniqueness and trace/span indexes.
- [ ] Implement validation independent of Gin or GORM.
- [ ] Implement transaction-scoped ingestion that creates the Trace on `trace_start`, appends child Spans, and stores LLM/Tool details.
- [ ] Make duplicate events return an idempotent accepted result without duplicate rows.
- [ ] Implement Trace state transitions so terminal states cannot return to `running`.
- [ ] Run focused tests and `go test ./...`.

## Task 4: Add HTTP ingestion and tenant-safe trace queries

**Files:** `internal/trace/handler.go`, `internal/http/router.go`, `internal/trace/integration_test.go`

- [ ] Write a failing integration test that posts an event with an Agent key, queries the Trace, and proves another tenant cannot read it.
- [ ] Run `go test ./internal/trace -run TestTraceTenantIsolation -count=1` and confirm it fails before routes exist.
- [ ] Add `POST /api/v1/ingest/events` with `202 Accepted` for accepted and duplicate events.
- [ ] Derive tenant and Agent identity from middleware context and pass them to the application service.
- [ ] Add `GET /api/v1/traces` with tenant-scoped pagination and Agent/status/risk filters.
- [ ] Add `GET /api/v1/traces/:traceId` and `GET /api/v1/traces/:traceId/spans` with tenant-scoped lookup.
- [ ] Return stable error codes such as `INVALID_EVENT`, `UNAUTHORIZED_AGENT`, `TRACE_NOT_FOUND`, and `TENANT_ACCESS_DENIED`.
- [ ] Run the integration test against containerized MySQL and all Go tests.

## Task 5: Add Redis Stream publishing and worker processing

**Files:** `internal/worker/stream.go`, `analysis.go`, `analysis_test.go`, `cmd/worker/main.go`

- [ ] Write failing worker tests for successful consumption, duplicate consumption, transient retry, and dead-letter after the maximum attempts.
- [ ] Run `go test ./internal/worker -run TestAnalysisWorker -count=1` and confirm it fails before the worker exists.
- [ ] Define a versioned stream message containing tenant, event, trace, and span IDs.
- [ ] Publish an analysis message after ingestion commits; track `pending`, `queued`, `success`, `retrying`, and `dead` states.
- [ ] Implement a consumer group with explicit acknowledgement only after successful idempotent processing.
- [ ] Update initial usage totals and Trace status in the worker; defer external LLM risk analysis until the event pipeline is proven.
- [ ] Add bounded exponential backoff and a dead-letter stream.
- [ ] Run worker tests with a disposable Redis container and `go test ./...`.

## Task 6: Build the first Next.js trace console

**Files:** `web/package.json`, `web/tsconfig.json`, `web/app/layout.tsx`, trace pages, trace components, `web/lib/api/types.ts`, `client.ts`, and `trace-table.test.tsx`

- [ ] Write a failing component test for rendering a trace row and a meaningful empty state.
- [ ] Run `npm test -- trace-table.test.tsx` from `web` and confirm it fails because the app does not exist.
- [ ] Create a strict TypeScript Next.js App Router app with typed TraceSummary, TraceDetail, and Span types matching the Go API.
- [ ] Implement a typed API client that handles non-2xx responses using stable backend error codes.
- [ ] Implement the paginated Trace list with filters, loading, error, empty, and unauthorized states.
- [ ] Implement the Trace detail page and recursive Span tree using `parentSpanId`.
- [ ] Run the focused frontend test, `npm test`, and `npm run build`.

## Task 7: Add audit logging and baseline security checks

**Files:** `internal/audit/model.go`, `repository.go`, Agent and Trace services, `internal/http/middleware/limits.go`, `internal/audit/audit_test.go`

- [ ] Write failing tests proving Agent creation and key rotation create append-only audit entries and oversized event payloads are rejected.
- [ ] Run the focused tests and confirm they fail before audit and limit code exists.
- [ ] Implement audit persistence with actor, tenant, action, resource, request ID, and before/after snapshots.
- [ ] Add request body limits, per-Agent ingestion rate limiting, and Authorization-header redaction in logs.
- [ ] Ensure audit records are tenant-scoped and never expose raw API keys.
- [ ] Run security-focused tests and all backend tests.

## Task 8: End-to-end verification and documentation

**Files:** `README.md`, `Makefile` or `Taskfile.yml`

- [ ] Add local setup commands for containers, migrations, API, worker, frontend, and tests.
- [ ] Add a documented curl flow that creates an Agent, posts four events, and queries the Trace.
- [ ] Run `go test ./...`, `go vet ./...`, frontend tests, frontend build, and the documented smoke flow.
- [ ] Re-read the design specification against implemented acceptance criteria and list intentionally deferred items.
- [ ] Only report implemented scope after fresh command output confirms the acceptance criteria.

## Verification commands

Backend:

```powershell
go test ./...
go vet ./...
```

Frontend:

```powershell
Set-Location web
npm test
npm run build
```

Local dependencies:

```powershell
docker compose up -d mysql redis
go run ./cmd/api
go run ./cmd/worker
```

The first implementation batch should stop after Task 4, review the API and data model, then continue with the worker and frontend.
