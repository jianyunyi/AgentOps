import { describe, expect, it } from "vitest";
import { AgentOpsClient } from "../src/client.js";

const secret = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE=";

describe("AgentOpsClient configuration", () => {
  it("rejects unsafe or non-absolute base URLs", () => {
    for (const baseUrl of ["/api", "ftp://example.com", "https://user:pass@example.com", "https://example.com?token=secret", "https://example.com#fragment"]) {
      expect(() => new AgentOpsClient({ baseUrl, apiKey: "ag_live_test", signingSecret: secret })).toThrow();
    }
  });

  it("requires a non-empty API key and a 32-byte Base64 signing secret", () => {
    expect(() => new AgentOpsClient({ baseUrl: "https://example.com", apiKey: " ", signingSecret: secret })).toThrow();
    expect(() => new AgentOpsClient({ baseUrl: "https://example.com", apiKey: "ag_live_test", signingSecret: "bad" })).toThrow();
    expect(() => new AgentOpsClient({ baseUrl: "https://example.com", apiKey: "ag_live_test", signingSecret: "YWJj" })).toThrow();
  });

  it("applies bounded retry defaults and accepts padded Base64", () => {
    const client = new AgentOpsClient({ baseUrl: "https://example.com/", apiKey: "ag_live_test", signingSecret: secret });
    expect(client).toBeDefined();
  });

  it("rejects invalid retry policy values", () => {
    expect(() => new AgentOpsClient({ baseUrl: "https://example.com", apiKey: "ag_live_test", signingSecret: secret, maxAttempts: 6 })).toThrow();
    expect(() => new AgentOpsClient({ baseUrl: "https://example.com", apiKey: "ag_live_test", signingSecret: secret, baseRetryDelayMs: 200, maxRetryDelayMs: 100 })).toThrow();
  });
});
