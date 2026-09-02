# AgentOps TypeScript SDK

Server-side Node.js client for securely ingesting AgentScope events with the frozen HMAC v1 protocol.

## Runtime boundary

This package is for Node.js 18+ server processes. Do not import it from a browser bundle or React Client Component. Never put `AGENT_API_KEY` or `AGENT_SIGNING_SECRET` in `NEXT_PUBLIC_*`, `localStorage`, cookies, URLs, source code, or logs.

## Install and build

```powershell
npm install @jianyunyi/agentops
```

For repository development:

```powershell
npm ci
npm run typecheck
npm test -- --run
npm run build
```

## Server-side usage

```ts
import { AgentOpsClient } from "@jianyunyi/agentops";

const client = new AgentOpsClient({
  baseUrl: process.env.AGENTOPS_BASE_URL!,
  apiKey: process.env.AGENT_API_KEY!,
  signingSecret: process.env.AGENT_SIGNING_SECRET!,
});

const result = await client.ingest({
  event_id: "evt_example",
  trace_id: "trace_example",
  span_id: "span_example",
  event_type: "llm_call",
  occurred_at: new Date().toISOString(),
  payload: { prompt: "redacted or already-safe content" },
});
console.log(result.duplicate);
```

The Signing Secret is the one-time Base64 value returned by Agent management after credential creation or rotation. Store it in a server secret manager. The SDK generates a timestamp and fresh nonce for every attempt, signs the exact serialized request body, and retries only transient failures (`429`, `5xx`, and transport failures). Caller cancellation is honored through `AbortSignal`.

The signature format is compatible with the AgentScope Go SDK: HMAC-SHA256 over `v1`, method, path, Unix timestamp, nonce, and SHA-256 Body hash. HMAC does not replace HTTPS.

## Example

Set `AGENTOPS_BASE_URL`, `AGENT_API_KEY`, and `AGENT_SIGNING_SECRET`, then run the source example from a server-side Node process after building the package.
