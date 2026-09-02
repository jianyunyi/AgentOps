import type { Agent } from "./types";
import { APIError } from "./client";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers({ "content-type": "application/json", ...init?.headers });
  const csrf = typeof document === "undefined" ? "" : document.cookie.split(";").map((item) => item.trim()).find((item) => item.startsWith("agentscope_csrf="))?.split("=")[1] ?? "";
  if (csrf && init?.method && init.method !== "GET") headers.set("X-CSRF-Token", decodeURIComponent(csrf));
  const response = await fetch(path, { ...init, credentials: "include", headers });
  if (response.status === 204) return undefined as T;
  const raw = await response.text();
  const body = (raw ? JSON.parse(raw) : {}) as { data?: T; error?: { code: string; message: string } };
  if (!response.ok || body.data === undefined) throw new APIError(body.error?.code ?? "REQUEST_FAILED", body.error?.message ?? "Request failed", response.status);
  return body.data;
}
export const listAgents = () => request<Agent[]>("/api/v1/agents");
export type AgentCredentialReveal = { agent: Agent; api_key: string; signing_secret: string };
export type CredentialMigrationStatus = {
  summary: { total_agents: number; migrated_agents: number; legacy_agents: number };
  agents: Array<{ id: string; name: string; environment: string; status: string; last_used_at?: string }>;
  pagination: { page: number; page_size: number; total: number };
};
export const createAgent = (input: { name: string; description: string; environment: string }) => request<AgentCredentialReveal>("/api/v1/agents", { method: "POST", body: JSON.stringify(input) });
export const rotateAgentKey = (id: string) => request<AgentCredentialReveal>(`/api/v1/agents/${encodeURIComponent(id)}/rotate-key`, { method: "POST" });
export const revokeAgentKey = (id: string) => request<null>(`/api/v1/agents/${encodeURIComponent(id)}/revoke-key`, { method: "POST" });
export const getAgentCredentialMigrationStatus = (params: { page?: number; page_size?: number; q?: string } = {}) => {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.page_size) search.set("page_size", String(params.page_size));
  if (params.q?.trim()) search.set("q", params.q.trim());
  const suffix = search.toString();
  return request<CredentialMigrationStatus>(`/api/v1/agents/migration-status${suffix ? `?${suffix}` : ""}`);
};
