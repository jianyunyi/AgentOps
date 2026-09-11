const MAX_ERROR_RESPONSE_BYTES = 64 * 1024;
const MAX_ERROR_MESSAGE_LENGTH = 512;

export class AgentOpsAPIError extends Error {
  public readonly statusCode: number;
  public readonly code: string;
  public readonly requestId: string;
  public readonly retryAfterMs: number;

  public constructor(statusCode: number, code: string, message: string, requestId: string, retryAfterMs: number) {
    super(message);
    this.name = "AgentOpsAPIError";
    this.statusCode = statusCode;
    this.code = code;
    this.requestId = requestId;
    this.retryAfterMs = retryAfterMs;
  }
}

export function parseAPIError(statusCode: number, headers: Headers, body: Buffer, forbiddenValues: string[]): AgentOpsAPIError {
  const boundedBody = body.subarray(0, MAX_ERROR_RESPONSE_BYTES).toString("utf8");
  let parsed: { error?: { code?: unknown; message?: unknown } } = {};
  try {
    parsed = JSON.parse(boundedBody) as typeof parsed;
  } catch {
    parsed = {};
  }
  const code = typeof parsed.error?.code === "string" ? truncate(parsed.error.code) : "";
  let message = typeof parsed.error?.message === "string" ? truncate(parsed.error.message) : "";
  if (!message) message = statusText(statusCode);
  message = redact(message, forbiddenValues);
  const requestId = sanitizeRequestId(headers.get("X-Request-ID") ?? "");
  return new AgentOpsAPIError(statusCode, code, message, requestId, parseRetryAfter(headers.get("Retry-After")));
}

function statusText(statusCode: number): string {
  const messages: Record<number, string> = { 400: "Bad Request", 401: "Unauthorized", 403: "Forbidden", 404: "Not Found", 409: "Conflict", 429: "Too Many Requests", 500: "Internal Server Error", 502: "Bad Gateway", 503: "Service Unavailable", 504: "Gateway Timeout" };
  return messages[statusCode] ?? "Request Failed";
}

function truncate(value: string): string {
  const normalized = value.trim().replace(/[\r\n]+/g, " ");
  return Array.from(normalized).slice(0, MAX_ERROR_MESSAGE_LENGTH).join("");
}

function redact(value: string, forbiddenValues: string[]): string {
  return forbiddenValues.filter(Boolean).reduce((result, forbidden) => result.split(forbidden).join("[REDACTED]"), value);
}

function sanitizeRequestId(value: string): string {
  const trimmed = value.trim();
  return trimmed.length <= 128 && !/[\r\n]/.test(trimmed) ? trimmed : "";
}

function parseRetryAfter(value: string | null): number {
  if (!value) return 0;
  const seconds = Number.parseInt(value.trim(), 10);
  if (Number.isInteger(seconds) && seconds >= 0) return seconds * 1000;
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : Math.max(0, timestamp - Date.now());
}
