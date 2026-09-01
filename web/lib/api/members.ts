import { request } from "./client";
import type { Member } from "./types";

export const listMembers = () => request<Member[]>("/api/v1/members");
export const changeMemberRole = (id: string, role: string) => request<undefined>(`/api/v1/members/${encodeURIComponent(id)}/role`, { method: "PATCH", body: JSON.stringify({ role }) });
export const disableMember = (id: string) => request<undefined>(`/api/v1/members/${encodeURIComponent(id)}/disable`, { method: "POST" });
export const transferOwner = (id: string) => request<undefined>(`/api/v1/members/${encodeURIComponent(id)}/transfer-owner`, { method: "POST" });
export const createInvitation = (email: string, role: string, ttlHours = 48) => request<{ id: string; email: string; role: string; status: string; expires_at: string; invite_token: string }>("/api/v1/members/invitations", { method: "POST", body: JSON.stringify({ email, role, ttl_hours: ttlHours }) });
