# AgentScope Production Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move AgentScope from Alpha to an internally usable Beta by adding tenant bootstrap, credential lifecycle, auditability, transactional outbox delivery, real risk detection, and authenticated management UI.

**Architecture:** Keep the modular Go monolith. User authentication owns tenant context and RBAC; Agent credentials own ingestion identity. All business mutations emit append-only audit records, and event analysis is delivered through a MySQL outbox publisher into Redis Streams. Next.js consumes the same authenticated API for login and Agent management.

**Tech Stack:** Go, Gin, GORM, MySQL, Redis Streams, bcrypt, Next.js, React, TypeScript, Vitest.

---

## Task 1: Tenant bootstrap and user registration

**Files:** `internal/tenant/model.go`, `internal/tenant/repository.go`, `internal/auth/service.go`, `internal/auth/handler.go`, `internal/auth/handler_test.go`, `internal/platform/database/migrate.go`

- [ ] Test that registration creates one tenant and an Owner user atomically.
- [ ] Test duplicate email rejection and password validation.
- [ ] Add tenant and user creation repository operations with a transaction boundary.
- [ ] Add `POST /api/v1/auth/register` returning a session cookie and tenant metadata.
- [ ] Ensure email is normalized and stored uniquely within the global user namespace.
- [ ] Migrate tenants and users before sessions; never log passwords.
- [ ] Run `go test ./...` and `go vet ./...`.

## Task 2: Agent credential lifecycle

**Files:** `internal/agent/model.go`, `repository.go`, `service.go`, `handler.go`, tests, `internal/http/router.go`

- [ ] Test key rotation revokes the previous credential and returns a new raw key once.
- [ ] Test revoked and expired keys cannot authenticate.
- [ ] Test only a tenant member with `agent:write` can rotate or revoke credentials.
- [ ] Implement credential rotation, revocation, optional expiry, and last-used timestamps.
- [ ] Add `POST /api/v1/agents/:id/rotate-key` and `POST /api/v1/agents/:id/revoke-key`.
- [ ] Enforce tenant-scoped Agent lookup and append audit events for every lifecycle mutation.
- [ ] Run focused tests and the full backend suite.

## Task 3: Append-only audit log

**Files:** `internal/audit/model.go`, `repository.go`, `service.go`, tests, Agent/Auth/Trace services

- [ ] Test audit records contain tenant, actor, action, resource, request ID, and before/after snapshots.
- [ ] Test raw API keys and passwords never appear in snapshots.
- [ ] Implement append-only audit persistence and tenant-scoped listing.
- [ ] Add audit writes for registration, Agent creation, key rotation/revocation, risk review, and member changes.
- [ ] Add `GET /api/v1/audit-logs` for Auditor/Owner roles.
- [ ] Run security-focused tests and the full backend suite.

## Task 4: Transactional Outbox and eventual delivery

**Files:** `internal/outbox/model.go`, `repository.go`, `publisher.go`, tests, `internal/trace/service.go`, `internal/platform/database/migrate.go`

- [ ] Test an ingested event commits its business row and outbox row in one transaction.
- [ ] Test a Redis outage leaves the outbox row pending and does not lose the event.
- [ ] Test successful publication marks an outbox row delivered; failures back off and eventually dead-letter.
- [ ] Move Redis publication out of the request transaction and into an outbox publisher loop.
- [ ] Add unique event keys and idempotent outbox claiming to support multiple publishers.
- [ ] Run tests with a disposable Redis instance when available and all backend tests otherwise.

## Task 5: Real Prompt-injection and sensitive-data detection

**Files:** `internal/risk/model.go`, `engine.go`, `rules.go`, `service.go`, tests, worker integration

- [ ] Test deterministic detection of instruction override, system-prompt extraction, credential, email, phone, and API-key patterns.
- [ ] Test risk aggregation chooses the highest severity and does not duplicate the same rule/span event.
- [ ] Implement rule engine with versioned rule codes and redacted evidence.
- [ ] Add an LLM analyzer interface with structured JSON output validation; failures fall back to rules-only results.
- [ ] Persist risk events and update Trace risk level idempotently.
- [ ] Never send raw secrets to the LLM analyzer; pass redacted content and bounded payloads.
- [ ] Run risk tests and worker tests.

## Task 6: Authenticated frontend login and Agent management

**Files:** `web/app/login/page.tsx`, `web/app/settings/agents/page.tsx`, components, API client/types, tests

- [ ] Test login form success, invalid-credential error, and session-required redirect behavior.
- [ ] Test Agent list rendering, creation modal, one-time API-key reveal, rotate, and revoke actions.
- [ ] Add typed auth and Agent API clients using `credentials: include`.
- [ ] Add protected dashboard layout behavior and explicit unauthorized states.
- [ ] Add Agent status and credential lifecycle UI without exposing stored hashes.
- [ ] Run all frontend tests and `npm run build`.

## Final verification

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; $env:GOMODCACHE='F:\AgentOps\.tmp-go-mod'; go test ./...; go vet ./...
Set-Location web; npm.cmd test; npm.cmd run build
```

Do not claim production readiness until the complete suite, migration startup, and documented smoke flow have been run successfully.
