# P1-7 TypeScript/Node Agent SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Node.js TypeScript SDK that signs and reliably ingests AgentScope events without exposing credentials to browser runtimes.

**Architecture:** Create an independent package under `sdk/typescript` with no dependency on the server or Next.js. Use Node built-in `node:crypto`, `fetch`, `AbortSignal`, and typed public modules for signing, retry classification, error handling, and HTTP transport. Serialize an event once, reuse the exact UTF-8 bytes for every attempt, and generate a fresh timestamp/nonce/signature per attempt.

**Tech Stack:** TypeScript, Node.js 18+, native Fetch/Web-compatible AbortSignal, Node `crypto`, Vitest, npm package metadata.

---

### Task 1: Package scaffold and public event/config types

**Files:**
- Create: `sdk/typescript/package.json`
- Create: `sdk/typescript/package-lock.json`
- Create: `sdk/typescript/tsconfig.json`
- Create: `sdk/typescript/src/types.ts`
- Create: `sdk/typescript/src/index.ts`
- Create: `sdk/typescript/test/config.test.ts`

- [ ] **Step 1: Write failing configuration tests**

Test `AgentOpsClient` construction requirements before the client exists: reject a relative URL, non-HTTP scheme, URL credentials, query or fragment; reject blank API key and invalid/non-32-byte Base64 signing secret; accept raw and padded Base64; apply defaults of 3 attempts, 100ms base delay, 10s maximum delay; reject attempts outside 1–5 and max delay below base delay.

- [ ] **Step 2: Run the focused test and confirm the expected failure**

Run from `sdk/typescript`:

```powershell
node_modules/.bin/vitest run test/config.test.ts
```

Expected result: FAIL because `src/client.ts` and the public configuration are not implemented.

- [ ] **Step 3: Add package metadata, lockfile, and typed contracts**

Create an ESM package with Node 18 engines, `build`, `test`, `typecheck`, and `test:integration` scripts, then generate `package-lock.json` from that manifest so CI can use `npm ci`. Use `typescript` and `vitest` only as dev dependencies. Define `AgentEvent` with the exact service JSON names (`event_id`, `trace_id`, `span_id`, `parent_span_id`, `event_type`, `sequence`, `occurred_at`, `payload`), `IngestResult`, retry options, `AgentOpsClientOptions`, and a typed `FetchLike` injection point.

- [ ] **Step 4: Implement minimal constructor validation**

Implement `AgentOpsClient` constructor validation in `src/client.ts` using `new URL`, accepting only absolute `http:` and `https:` URLs with no username, password, query, or fragment. Trim the base path, require non-empty credentials, decode a 32-byte Base64 secret using Node `Buffer`, and normalize retry defaults. Keep the secret in a private `Buffer` and never expose it in public properties.

- [ ] **Step 5: Run the focused test and typecheck**

Run:

```powershell
npm test -- --run test/config.test.ts
npm run typecheck
```

Expected result: configuration tests pass and TypeScript exits with code 0.

### Task 2: Frozen HMAC v1 signing and nonce generation

**Files:**
- Create: `sdk/typescript/src/signing.ts`
- Create: `sdk/typescript/test/signing.test.ts`
- Modify: `sdk/typescript/src/index.ts`

- [ ] **Step 1: Write failing protocol-vector tests**

Use secret `01234567890123456789012345678901`, Body `hello`, method `POST`, path `/api/v1/ingest/events`, timestamp `1700000000`, nonce `nonce-fixed`. Assert the body SHA-256, canonical string, and exact signature `v1=f7151e596375c3562ff93158cd2dd289b4a17bdf23f5b8149902b8bb8f4b3cb3`. Assert generated nonces are distinct, lowercase hexadecimal, and 64 characters.

- [ ] **Step 2: Run the signing test and confirm the expected failure**

Run `npm test -- --run test/signing.test.ts`. Expected result: FAIL because signing helpers are missing.

- [ ] **Step 3: Implement the minimal signing module**

Implement:

```ts
export function hashBody(body: Buffer): string;
export function canonicalRequest(method: string, path: string, timestamp: number, nonce: string, bodyHash: string): string;
export function signRequest(secret: Buffer, method: string, path: string, body: Buffer, timestamp: number, nonce: string): string;
export function newNonce(): string;
```

Use `createHash('sha256')`, `createHmac('sha256')`, lowercase hex, and the exact six-line canonical order `v1`, method, path, timestamp, nonce, body hash. Generate 32 random bytes using `randomBytes`.

- [ ] **Step 4: Run the focused signing test**

Run `npm test -- --run test/signing.test.ts`. Expected result: all protocol vector and nonce tests pass.

### Task 3: Safe API errors and cancellable retry policy

**Files:**
- Create: `sdk/typescript/src/errors.ts`
- Create: `sdk/typescript/src/retry.ts`
- Create: `sdk/typescript/test/errors.test.ts`
- Create: `sdk/typescript/test/retry.test.ts`
- Modify: `sdk/typescript/src/types.ts`
- Modify: `sdk/typescript/src/index.ts`

- [ ] **Step 1: Write failing error and retry tests**

Test that API JSON `{error:{code,message}}` becomes `AgentOpsAPIError` with status, code, bounded single-line message, sanitized request ID, and parsed `Retry-After`; malformed/non-JSON errors use a stable fallback. Include API key, Signing Secret, and raw request Body in a server error message and assert all are redacted. Test retry classification for network errors, 429, 500, 502, 503, 504, and no retry for 400, 401, 403, 404. Test delay caps and an AbortSignal cancelling the delay.

- [ ] **Step 2: Run the tests and confirm the expected failure**

Run:

```powershell
npm test -- --run test/errors.test.ts test/retry.test.ts
```

Expected result: FAIL because error and retry modules are missing.

- [ ] **Step 3: Implement bounded error parsing and redaction**

Create `AgentOpsAPIError` with readonly `statusCode`, `code`, `message`, `requestId`, and `retryAfterMs`. Parse at most 64 KiB, trim and normalize line breaks, cap messages at 512 Unicode code points, accept only safe request IDs up to 128 characters, and redact all configured forbidden values before returning the error. Do not store the request Body in the error.

- [ ] **Step 4: Implement retry classification and context cancellation**

Implement `isRetryableError`, `retryDelayMs`, and `waitForRetry(signal, delayMs)`. Use `Retry-After` when present but cap it at the configured maximum; otherwise use bounded exponential backoff. Abort immediately when the signal is already aborted or aborts while waiting.

- [ ] **Step 5: Run the focused error/retry tests**

Run `npm test -- --run test/errors.test.ts test/retry.test.ts`. Expected result: all error and retry tests pass without printing credentials.

### Task 4: HTTP ingest client and retry loop

**Files:**
- Create: `sdk/typescript/src/client.ts`
- Create: `sdk/typescript/test/client.test.ts`
- Modify: `sdk/typescript/src/index.ts`

- [ ] **Step 1: Write failing HTTP behavior tests**

Use an injected `fetch` implementation or local HTTP server to assert: POST path `/api/v1/ingest/events`; `Authorization`, timestamp, nonce, signature, request ID, content type, and user agent headers; exact JSON Body reuse across retries; fresh nonce/signature each retry; duplicate response parsing; 503 retry success; 400/401 no retry; oversized response rejection; malformed success rejection; and AbortSignal propagation.

- [ ] **Step 2: Run the HTTP tests and confirm the expected failure**

Run `npm test -- --run test/client.test.ts`. Expected result: FAIL because `ingest` is not implemented.

- [ ] **Step 3: Implement one signed HTTP attempt**

Serialize `AgentEvent` once with `JSON.stringify` and `Buffer.from(body, 'utf8')`. Build the URL from the validated base URL and fixed ingest path, generate timestamp/nonce/request ID, sign the exact path and Body, call the injected or global `fetch`, cap response reading at 64 KiB, and parse `{data:{duplicate}}` or a safe API error. Never log or attach credentials to thrown errors.

- [ ] **Step 4: Implement the retry loop**

Retry only the classifications from Task 3 until `maxAttempts`; create fresh auth headers on every attempt while reusing the same Body bytes. Stop immediately for non-retryable errors, caller cancellation, or deadline cancellation. Return `IngestResult` on the first accepted response.

- [ ] **Step 5: Run focused and full package tests**

Run:

```powershell
npm test -- --run test/client.test.ts
npm test -- --run
npm run typecheck
npm run build
```

Expected result: all package tests pass, typecheck passes, and `dist` is generated without source/test files.

### Task 5: Example, documentation, and optional integration test

**Files:**
- Create: `sdk/typescript/examples/ingest.ts`
- Create: `sdk/typescript/test/integration.test.ts`
- Create: `sdk/typescript/README.md`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add a safe Node server example**

Read `AGENTOPS_BASE_URL`, `AGENT_API_KEY`, and `AGENT_SIGNING_SECRET` from the environment, construct the client, ingest a unique event, and print only the duplicate flag. Do not include fallback credentials or print environment values.

- [ ] **Step 2: Add opt-in real API integration coverage**

When `AGENTSCOPE_INTEGRATION=1` and the three credential variables are present, call the real configured API with a unique event and assert a successful accepted response. Skip with a clear message when credentials are absent. The test must not create or rotate credentials and must not print the secret values.

- [ ] **Step 3: Document server-only usage and migration requirements**

Document Node 18+, installation/build commands, environment configuration, a minimal server-side example, retry semantics, HMAC protocol compatibility with the Go SDK, one-time credentials, and an explicit warning that the package must not be bundled into browser code or used with `NEXT_PUBLIC_*` values.

- [ ] **Step 4: Add CI package gates**

Add a CI job or steps that run `npm ci`, `npm run typecheck`, `npm test -- --run`, `npm run build`, and `npm audit --audit-level=high` in `sdk/typescript`. Run the opt-in integration test only in the existing integration job when all SDK credential secrets are configured.

### Task 6: Final verification and delivery

**Files:**
- Modify: `docs/superpowers/plans/2026-09-02-p1-7-typescript-agent-sdk.md` to check completed steps only after verification.

- [ ] **Step 1: Run complete backend regression checks**

Run from repository root:

```powershell
$env:GOCACHE='F:\AgentOps\.tmp-go-cache'; go test ./... -count=1; go vet ./...
```

Expected result: exit code 0 with no test failures or vet diagnostics.

- [ ] **Step 2: Run complete TypeScript SDK checks**

Run from `sdk/typescript`:

```powershell
npm ci
npm run typecheck
npm test -- --run
npm run build
```

If the local npm installation is unavailable, use the bundled Node runtime only with already-present dependencies and record the environment limitation; do not weaken package security or commit generated dependency directories.

- [ ] **Step 3: Run real integration checks when configured**

With MySQL `127.0.0.1:3307`, Redis `127.0.0.1:3509`, and opt-in SDK credentials, run the existing backend integration suite and SDK integration test. Expected result: all configured tests pass; credential-free tests remain deterministic and skipped only where specified.

- [ ] **Step 4: Inspect security and repository state**

Run `git diff --check`, inspect staged files, search the SDK for request-body logging and credential persistence, confirm no real secret appears in examples/docs, and confirm `git status --short` contains only intended files.

- [ ] **Step 5: Commit and push**

Create a focused commit:

```powershell
git add sdk/typescript .github/workflows/ci.yml
git commit -m "feat: add typescript agent sdk"
git push origin main
```

Only report completion after the commit is pushed and fresh verification output is available.
