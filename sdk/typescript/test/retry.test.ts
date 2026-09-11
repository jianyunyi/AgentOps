import { describe, expect, it } from "vitest";
import { isRetryableStatus, retryDelayMs, waitForRetry } from "../src/retry.js";

describe("AgentOps retry policy", () => {
  it("retries transient statuses only", () => {
    for (const status of [429, 500, 502, 503, 504]) expect(isRetryableStatus(status)).toBe(true);
    for (const status of [400, 401, 403, 404, 409]) expect(isRetryableStatus(status)).toBe(false);
  });

  it("uses capped exponential delay and Retry-After", () => {
    const policy = { maxAttempts: 3, baseRetryDelayMs: 100, maxRetryDelayMs: 500 };
    expect(retryDelayMs(policy, 1, 0)).toBe(100);
    expect(retryDelayMs(policy, 2, 0)).toBe(300);
    expect(retryDelayMs(policy, 2, 1000)).toBe(500);
  });

  it("honors AbortSignal while waiting", async () => {
    const controller = new AbortController();
    const waiting = waitForRetry(controller.signal, 1000);
    controller.abort();
    await expect(waiting).rejects.toMatchObject({ name: "AbortError" });
  });
});
