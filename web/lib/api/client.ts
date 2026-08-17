import type { Paginated, Span, TraceDetail, TraceSummary } from "./types";

export class APIError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, credentials: "include" });
  const body = (await response.json()) as { data?: T; error?: { code: string; message: string } };
  if (!response.ok || !body.data) {
    throw new APIError(body.error?.code ?? "UNKNOWN_ERROR", body.error?.message ?? "Request failed", response.status);
  }
  return body.data;
}

export function listTraces(page = 1, pageSize = 20): Promise<Paginated<TraceSummary>> {
  return request<Paginated<TraceSummary>>(`/api/v1/traces?page=${page}&page_size=${pageSize}`);
}

export async function getTrace(traceId: string): Promise<TraceDetail> {
  const trace = await request<Omit<TraceDetail, "spans">>(`/api/v1/traces/${encodeURIComponent(traceId)}`);
  const spans = await request<Span[]>(`/api/v1/traces/${encodeURIComponent(traceId)}/spans`);
  return { ...trace, spans };
}
