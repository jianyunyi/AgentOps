import { request } from "./client";
import type { AuditRecord, Paginated, RiskEvent } from "./types";

export const listAuditLogs = (page = 1): Promise<Paginated<AuditRecord>> => request<Paginated<AuditRecord>>(`/api/v1/audit-logs?page=${page}&page_size=20`);
export const listRiskEvents = (page = 1, status = ""): Promise<Paginated<RiskEvent>> => request<Paginated<RiskEvent>>(`/api/v1/risk-events?page=${page}&page_size=20${status ? `&status=${encodeURIComponent(status)}` : ""}`);
export const reviewRiskEvent = (id: string, status: string): Promise<null> => request<null>(`/api/v1/risk-events/${encodeURIComponent(id)}/review`, { method: "POST", body: JSON.stringify({ status }) });
