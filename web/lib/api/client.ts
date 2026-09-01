import type { Paginated, Span, TraceDetail, TraceSummary } from "./types";

export class APIError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(message);
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  const csrf = typeof document === "undefined" ? "" : document.cookie.split(";").map((item) => item.trim()).find((item) => item.startsWith("agentscope_csrf="))?.split("=")[1] ?? "";
  if (csrf && init?.method && init.method !== "GET") headers.set("X-CSRF-Token", decodeURIComponent(csrf));
  const response = await fetch(path, { ...init, headers, credentials: "include" });
  if (response.status === 204) return undefined as T;
  const raw = await response.text();
  const body = (raw ? JSON.parse(raw) : {}) as { data?: T; error?: { code: string; message: string } };
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
