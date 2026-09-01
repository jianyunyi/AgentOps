# AgentScope Phase 2 Security and Stability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add the minimum production security and operational controls for API traffic without changing business semantics.

**Architecture:** Add composable Gin middleware for request identity, body limits, CORS, and Redis-backed fixed-window rate limiting. Add explicit liveness/readiness/metrics endpoints with injected health checks so tests do not require infrastructure.

**Tech Stack:** Go, Gin, Redis v9, MySQL/GORM, atomic counters, HTTP middleware.

---

### Task 1: Request identity and HTTP safety middleware

**Files:** Create `internal/http/middleware.go`, test `internal/http/middleware_test.go`; modify `internal/http/router.go`, `cmd/api/main.go`, `internal/platform/config/config.go`.

- [ ] Test request ID propagation, body limit rejection, configured CORS origin, and disallowed origin.
- [ ] Implement middleware with `X-Request-ID`, `http.MaxBytesReader`, explicit CORS headers, and `OPTIONS` handling.
- [ ] Wire middleware before application routes and expose `WEB_ORIGIN` configuration.

### Task 2: Redis fixed-window rate limiting

**Files:** Create `internal/platform/ratelimit/redis.go`, test `internal/platform/ratelimit/redis_test.go`; modify router and config.

- [ ] Test the limiter allows requests under the limit and rejects after the limit.
- [ ] Implement atomic Redis `INCR` plus first-write `EXPIRE` with a stable key prefix and `Retry-After`.
- [ ] Apply conservative limits to login, registration, Agent ingest, and management writes.

### Task 3: Health and metrics endpoints

**Files:** Create `internal/platform/metrics/metrics.go`, test `internal/platform/metrics/metrics_test.go`; modify router and main.

- [ ] Test liveness, readiness dependency failures, and metrics output.
- [ ] Implement atomic request/error counters and a plain Prometheus-compatible exposition endpoint.
- [ ] Wire MySQL `PingContext` and Redis `Ping` into readiness.

### Task 4: Cookie and session hardening

**Files:** Modify `internal/auth/handler.go`, tests.

- [ ] Test session cookies include `HttpOnly`, `Secure`, and `SameSite=Lax`.
- [ ] Replace implicit Gin cookie serialization with `http.SetCookie` and add logout.

### Task 5: Verification and delivery

- [ ] Run `go test ./...`, `go vet ./...`, frontend tests, and frontend build.
- [ ] Run integration tests when Docker dependencies are available.
- [ ] Commit and push the phase-2 changes.
