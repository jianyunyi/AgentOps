import { listAgents } from "./agents";
import { listRiskEvents } from "./governance";
import { listTraces } from "./client";
import type { Agent, RiskEvent, TraceSummary } from "./types";

export interface OverviewData {
  agents: Agent[];
  traces: TraceSummary[];
  risks: RiskEvent[];
}

export async function getOverviewData(): Promise<OverviewData> {
  const [agents, traces, risks] = await Promise.all([
    listAgents(),
    listTraces(1, 5),
    listRiskEvents(1, "open"),
  ]);

  return { agents, traces: traces.data, risks: risks.data };
}
