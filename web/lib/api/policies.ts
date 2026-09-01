import { request } from "./client";
import type { Policy } from "./types";

export const listPolicies = () => request<Policy[]>("/api/v1/policies");
export const createPolicy = (input: { name: string; rules_enabled: boolean; llm_enabled: boolean; max_input_bytes: number }) => request<Policy>("/api/v1/policies", { method: "POST", body: JSON.stringify(input) });
export const activatePolicy = (id: string) => request<undefined>(`/api/v1/policies/${encodeURIComponent(id)}/activate`, { method: "POST" });
