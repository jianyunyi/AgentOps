import type { RetryPolicy } from "./types.js";
import { AgentOpsAPIError } from "./errors.js";

export function isRetryableStatus(statusCode: number): boolean {
  return statusCode === 408 || statusCode === 429 || statusCode === 500 || statusCode === 502 || statusCode === 503 || statusCode === 504;
}

export function isRetryableError(error: unknown): boolean {
  return error instanceof AgentOpsAPIError ? isRetryableStatus(error.statusCode) : error instanceof TypeError;
}

export function retryDelayMs(policy: Required<RetryPolicy>, failureAttempt: number, retryAfterMs: number): number {
  if (retryAfterMs > 0) return Math.min(retryAfterMs, policy.maxRetryDelayMs);
  let delay = policy.baseRetryDelayMs;
  for (let index = 1; index < failureAttempt; index += 1) delay = Math.min(policy.maxRetryDelayMs, delay * 3);
  return Math.min(delay, policy.maxRetryDelayMs);
}

export function waitForRetry(signal: AbortSignal | undefined, delayMs: number): Promise<void> {
  if (signal?.aborted) return Promise.reject(abortError());
  if (delayMs <= 0) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, delayMs);
    const onAbort = () => {
      clearTimeout(timer);
      signal?.removeEventListener("abort", onAbort);
      reject(abortError());
    };
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

function abortError(): DOMException {
  return new DOMException("The operation was aborted", "AbortError");
}
