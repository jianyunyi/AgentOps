import { AgentOpsAPIError, parseAPIError } from "./errors.js";
import { isRetryableError, retryDelayMs, waitForRetry } from "./retry.js";
import { newNonce, signRequest } from "./signing.js";
import type { AgentEvent, AgentOpsClientOptions, FetchLike, IngestOptions, IngestResult, RetryPolicy } from "./types.js";

const DEFAULT_MAX_ATTEMPTS = 3;
const DEFAULT_BASE_RETRY_DELAY_MS = 100;
const DEFAULT_MAX_RETRY_DELAY_MS = 10_000;
const INGEST_PATH = "/api/v1/ingest/events";
const MAX_RESPONSE_BODY_BYTES = 64 * 1024;

export class AgentOpsClient {
  private readonly baseUrl: URL;
  private readonly apiKey: string;
  private readonly signingSecret: Buffer;
  private readonly retry: Required<RetryPolicy>;
  private readonly fetcher: FetchLike;
  private readonly userAgent: string;

  public constructor(options: AgentOpsClientOptions) {
    this.baseUrl = parseBaseUrl(options.baseUrl);
    this.apiKey = requireText(options.apiKey, "agent API key is required");
    this.signingSecret = decodeSigningSecret(options.signingSecret);
    const maxAttempts = options.maxAttempts ?? DEFAULT_MAX_ATTEMPTS;
    const baseRetryDelayMs = options.baseRetryDelayMs ?? DEFAULT_BASE_RETRY_DELAY_MS;
    const maxRetryDelayMs = options.maxRetryDelayMs ?? DEFAULT_MAX_RETRY_DELAY_MS;
    if (!Number.isInteger(maxAttempts) || maxAttempts < 1 || maxAttempts > 5) {
      throw new Error("retry max attempts must be between 1 and 5");
    }
    if (!Number.isFinite(baseRetryDelayMs) || baseRetryDelayMs <= 0 || !Number.isFinite(maxRetryDelayMs) || maxRetryDelayMs <= 0 || maxRetryDelayMs < baseRetryDelayMs) {
      throw new Error("retry delays must be positive and max delay must not be less than base delay");
    }
    this.retry = { maxAttempts, baseRetryDelayMs, maxRetryDelayMs };
    this.fetcher = options.fetch ?? globalThis.fetch;
    if (typeof this.fetcher !== "function") {
      throw new Error("global fetch is unavailable; provide a fetch implementation");
    }
    this.userAgent = requireText(options.userAgent ?? "agentops-typescript-sdk/0.1", "user agent is required");
  }

  public async ingest(event: AgentEvent, options?: IngestOptions): Promise<IngestResult> {
    validateEvent(event);
    let body: Buffer;
    try {
      body = Buffer.from(JSON.stringify(event), "utf8");
    } catch {
      throw new Error("agent event cannot be serialized");
    }
    for (let attempt = 1; attempt <= this.retry.maxAttempts; attempt += 1) {
      try {
        return await this.ingestAttempt(body, options?.signal);
      } catch (error) {
        if (attempt === this.retry.maxAttempts || options?.signal?.aborted || !isRetryableError(error)) throw error;
        await waitForRetry(options?.signal, retryDelayMs(this.retry, attempt, error instanceof AgentOpsAPIError ? error.retryAfterMs : 0));
      }
    }
    throw new Error("agent ingest attempts exhausted");
  }

  private async ingestAttempt(body: Buffer, signal?: AbortSignal): Promise<IngestResult> {
    const endpoint = new URL(this.baseUrl.toString());
    endpoint.pathname = `${this.baseUrl.pathname.replace(/\/+$/, "")}${INGEST_PATH}`;
    endpoint.search = "";
    endpoint.hash = "";
    const timestamp = Math.floor(Date.now() / 1000);
    const nonce = newNonce();
    const requestId = `req_${newNonce()}`;
    const headers = new Headers({
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.apiKey}`,
      "X-Agent-Timestamp": String(timestamp),
      "X-Agent-Nonce": nonce,
      "X-Agent-Signature": signRequest(this.signingSecret, "POST", endpoint.pathname, body, timestamp, nonce),
      "X-Request-ID": requestId,
      "User-Agent": this.userAgent,
    });
    let response: Response;
    try {
      response = await this.fetcher(endpoint.toString(), { method: "POST", headers, body: body as unknown as BodyInit, signal });
    } catch (error) {
      throw error;
    }
    const responseBody = await readResponseBody(response);
    if (responseBody === null) {
      throw new AgentOpsAPIError(response.status, "RESPONSE_TOO_LARGE", "agent response is too large", requestId, 0);
    }
    if (!response.ok) {
      throw parseAPIError(response.status, response.headers, responseBody, [this.apiKey, body.toString("utf8"), this.signingSecret.toString("base64"), this.signingSecret.toString("base64").replace(/=+$/, "")]);
    }
    let envelope: { data?: { duplicate?: unknown } };
    try {
      envelope = JSON.parse(responseBody.toString("utf8")) as typeof envelope;
    } catch {
      throw new AgentOpsAPIError(response.status, "INVALID_RESPONSE", "agent response is invalid", requestId, 0);
    }
    if (!envelope.data || typeof envelope.data.duplicate !== "boolean") {
      throw new AgentOpsAPIError(response.status, "INVALID_RESPONSE", "agent response is invalid", requestId, 0);
    }
    return { duplicate: envelope.data.duplicate };
  }
}

function validateEvent(event: AgentEvent): void {
  if (!event || !event.event_id?.trim() || !event.trace_id?.trim() || !event.span_id?.trim() || !event.event_type?.trim() || !event.occurred_at?.trim()) {
    throw new Error("event id, trace id, span id, event type, and occurred at are required");
  }
  if (Number.isNaN(Date.parse(event.occurred_at))) throw new Error("event occurred at must be a valid timestamp");
}

async function readResponseBody(response: Response): Promise<Buffer | null> {
  if (!response.body) {
    const bytes = new Uint8Array(await response.arrayBuffer());
    return bytes.byteLength > MAX_RESPONSE_BODY_BYTES ? null : Buffer.from(bytes);
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const next = await reader.read();
      if (next.done) break;
      total += next.value.byteLength;
      if (total > MAX_RESPONSE_BODY_BYTES) {
        await reader.cancel();
        return null;
      }
      chunks.push(next.value);
    }
  } finally {
    reader.releaseLock();
  }
  return Buffer.concat(chunks.map((chunk) => Buffer.from(chunk)));
}

function parseBaseUrl(raw: string): URL {
  let parsed: URL;
  try {
    parsed = new URL(raw.trim());
  } catch {
    throw new Error("base URL must be an absolute HTTP(S) URL without credentials, query, or fragment");
  }
  if ((parsed.protocol !== "http:" && parsed.protocol !== "https:") || parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error("base URL must be an absolute HTTP(S) URL without credentials, query, or fragment");
  }
  parsed.pathname = parsed.pathname.replace(/\/+$/, "");
  return parsed;
}

function requireText(value: string, message: string): string {
  const trimmed = value.trim();
  if (!trimmed) throw new Error(message);
  return trimmed;
}

function decodeSigningSecret(raw: string): Buffer {
  const compact = raw.trim().replace(/=+$/, "");
  if (!compact || !/^[A-Za-z0-9+/]*$/.test(compact) || compact.length % 4 === 1) {
    throw new Error("agent signing secret must be a Base64-encoded 32-byte value");
  }
  const secret = Buffer.from(compact, "base64");
  if (secret.length !== 32 || secret.toString("base64").replace(/=+$/, "") !== compact) {
    throw new Error("agent signing secret must be a Base64-encoded 32-byte value");
  }
  return Buffer.from(secret);
}
