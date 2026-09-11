import { describe, expect, it } from "vitest";
import { AgentOpsAPIError, parseAPIError } from "../src/errors.js";

describe("AgentOps API errors", () => {
  it("parses bounded structured errors and Retry-After", () => {
    const error = parseAPIError(503, new Headers({ "X-Request-ID": "req_safe", "Retry-After": "2" }), Buffer.from(JSON.stringify({ error: { code: "TEMPORARY", message: "try again" } })), []);
    expect(error).toBeInstanceOf(AgentOpsAPIError);
    expect(error.statusCode).toBe(503);
    expect(error.code).toBe("TEMPORARY");
    expect(error.message).toContain("try again");
    expect(error.requestId).toBe("req_safe");
    expect(error.retryAfterMs).toBe(2000);
  });

  it("falls back safely for malformed responses and redacts forbidden values", () => {
    const apiKey = "ag_live_test_secret";
    const signingSecret = "c2lnbmluZ19zZWNyZXRfMTIz";
    const body = "{\"prompt\":\"private\"}";
    const error = parseAPIError(400, new Headers({ "X-Request-ID": "x".repeat(129) }), Buffer.from(`${apiKey} ${signingSecret} ${body}`), [apiKey, signingSecret, body]);
    expect(error.code).toBe("");
    expect(error.message).toContain("Bad Request");
    expect(error.message).not.toContain(apiKey);
    expect(error.message).not.toContain(signingSecret);
    expect(error.message).not.toContain(body);
    expect(error.requestId).toBe("");
  });

  it("caps server-controlled error messages", () => {
    const error = parseAPIError(500, new Headers(), Buffer.from(JSON.stringify({ error: { message: "x".repeat(1000) } })), []);
    expect(error.message.length).toBeLessThanOrEqual(512);
  });
});
