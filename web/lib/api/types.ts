export type TraceStatus = "running" | "success" | "failed" | "timeout";
export type RiskLevel = "none" | "low" | "medium" | "high" | "critical";

export interface TraceSummary {
  traceId: string;
  agentName: string;
  status: TraceStatus;
  riskLevel: RiskLevel;
  durationMs: number;
  totalTokens: number;
  estimatedCost: number;
  startedAt: string;
}

export interface Span {
  spanId: string;
  parentSpanId: string | null;
  spanType: string;
  name: string;
  status: string;
  sequence: number;
  inputSnapshot?: unknown;
  outputSnapshot?: unknown;
  durationMs: number;
  startedAt: string;
}

export interface TraceDetail extends TraceSummary {
  tenantId: string;
  agentId: string;
  endedAt?: string | null;
  spans: Span[];
}

export interface Paginated<T> {
  data: T[];
  pagination: { page: number; page_size: number; total: number };
}

export interface Agent {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  environment: string;
  status: string;
}

export interface CurrentUser {
  user_id: string;
  tenant_id: string;
  role: string;
  permissions: string[];
}
