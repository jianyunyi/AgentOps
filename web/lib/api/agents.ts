import type { Agent } from "./types";
import { APIError } from "./client";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, credentials: "include", headers: { "content-type": "application/json", ...init?.headers } });
  const body = (await response.json()) as { data?: T; error?: { code: string; message: string } };
  if (!response.ok || body.data === undefined) throw new APIError(body.error?.code ?? "REQUEST_FAILED", body.error?.message ?? "Request failed", response.status);
  return body.data;
}
export const listAgents = () => request<Agent[]>("/api/v1/agents");
export const createAgent = (input: { name: string; description: string; environment: string }) => request<{ agent: Agent; api_key: string }>("/api/v1/agents", { method: "POST", body: JSON.stringify(input) });
export const rotateAgentKey = (id: string) => request<{ agent: Agent; api_key: string }>(`/api/v1/agents/${encodeURIComponent(id)}/rotate-key`, { method: "POST" });
export const revokeAgentKey = (id: string) => request<null>(`/api/v1/agents/${encodeURIComponent(id)}/revoke-key`, { method: "POST" });
