import { request } from "./client";
import type { CurrentUser } from "./types";

export function login(email: string, password: string): Promise<{ user_id: string; tenant_id: string }> {
	return request<{ user_id: string; tenant_id: string }>("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
}

export function getCurrentUser(): Promise<CurrentUser> {
  return request<CurrentUser>("/api/v1/auth/me");
}
