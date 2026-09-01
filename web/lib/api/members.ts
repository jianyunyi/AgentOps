import { request } from "./client";
import type { Member } from "./types";

export const listMembers = () => request<Member[]>("/api/v1/members");
export const changeMemberRole = (id: string, role: string) => request<undefined>(`/api/v1/members/${encodeURIComponent(id)}/role`, { method: "PATCH", body: JSON.stringify({ role }) });
export const disableMember = (id: string) => request<undefined>(`/api/v1/members/${encodeURIComponent(id)}/disable`, { method: "POST" });
