export type AgentEvent = {
  event_id: string;
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  event_type: string;
  sequence?: number;
  occurred_at: string;
  payload: unknown;
};

export type IngestResult = {
  duplicate: boolean;
};

export type RetryPolicy = {
  maxAttempts?: number;
  baseRetryDelayMs?: number;
  maxRetryDelayMs?: number;
};

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export type AgentOpsClientOptions = RetryPolicy & {
  baseUrl: string;
  apiKey: string;
  signingSecret: string;
  fetch?: FetchLike;
  userAgent?: string;
};

export type IngestOptions = {
  signal?: AbortSignal;
};
