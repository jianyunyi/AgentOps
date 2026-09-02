import { describe, expect, it } from "vitest";
import { AgentOpsAPIError } from "../src/errors.js";
import { AgentOpsClient } from "../src/client.js";
import { signRequest } from "../src/signing.js";

const rawSecret = "01234567890123456789012345678901";
const signingSecret = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=";
const event = { event_id: "evt_1", trace_id: "trace_1", span_id: "span_1", event_type: "llm_call", occurred_at: "2026-09-02T00:00:00.000Z", payload: { prompt: "hello" } };

function client(fetcher: typeof fetch, options: Partial<ConstructorParameters<typeof AgentOpsClient>[0]> = {}) {
  return new AgentOpsClient({ baseUrl: "https://example.com/", apiKey: "ag_live_test", signingSecret, maxAttempts: 1, fetch: fetcher, ...options });
}

describe("AgentOps HTTP client", () => {
  it("signs the exact request body and parses duplicate", async () => {
    const fetcher = async (input: string | URL | Request, init?: RequestInit) => {
      const requestURL = String(input);
      const body = Buffer.from(init?.body as Uint8Array);
      const headers = new Headers(init?.headers);
      const timestamp = Number(headers.get("X-Agent-Timestamp"));
      expect(requestURL).toBe("https://example.com/api/v1/ingest/events");
      expect(headers.get("Authorization")).toBe("Bearer ag_live_test");
      expect(headers.get("X-Agent-Signature")).toBe(signRequest(Buffer.from(rawSecret), "POST", "/api/v1/ingest/events", body, timestamp, headers.get("X-Agent-Nonce")!));
      expect(headers.get("X-Request-ID")).toMatch(/^req_[0-9a-f]{64}$/);
      return new Response(JSON.stringify({ data: { duplicate: true } }), { status: 202, headers: { "content-type": "application/json" } });
    };
    await expect(client(fetcher).ingest(event)).resolves.toEqual({ duplicate: true });
  });

  it("retries transient errors with stable body and fresh nonce", async () => {
    const bodies: string[] = [];
    const nonces: string[] = [];
    let calls = 0;
    const fetcher = async (_input: string | URL | Request, init?: RequestInit) => {
      bodies.push(Buffer.from(init?.body as Uint8Array).toString("utf8"));
      nonces.push(new Headers(init?.headers).get("X-Agent-Nonce")!);
      calls += 1;
      if (calls === 1) return new Response("", { status: 503 });
      return new Response(JSON.stringify({ data: { duplicate: false } }), { status: 202 });
    };
    const result = await client(fetcher, { maxAttempts: 2, baseRetryDelayMs: 1, maxRetryDelayMs: 1 }).ingest(event);
    expect(result).toEqual({ duplicate: false });
    expect(bodies[0]).toBe(bodies[1]);
    expect(nonces[0]).not.toBe(nonces[1]);
  });

  it("does not retry non-transient API errors", async () => {
    let calls = 0;
    const fetcher = async () => { calls += 1; return new Response(JSON.stringify({ error: { code: "INVALID", message: "bad request" } }), { status: 400 }); };
    await expect(client(fetcher, { maxAttempts: 3, baseRetryDelayMs: 1, maxRetryDelayMs: 1 }).ingest(event)).rejects.toBeInstanceOf(AgentOpsAPIError);
    expect(calls).toBe(1);
  });

  it("rejects oversized responses without returning their content", async () => {
    const fetcher = async () => new Response("x".repeat(70 * 1024), { status: 503 });
    await expect(client(fetcher).ingest(event)).rejects.toMatchObject({ code: "RESPONSE_TOO_LARGE" });
  });
});
