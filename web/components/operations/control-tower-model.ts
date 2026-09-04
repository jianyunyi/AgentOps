import type { OverviewData } from "../../lib/api/overview";

export type SectorTone = "healthy" | "degraded" | "critical";

export interface SectorNode {
  id: string;
  label: string;
  callsign: string;
  tone: SectorTone;
  x: number;
  y: number;
  z: number;
}

export interface SectorRoute {
  traceId: string;
  fromId: string;
  toId: string;
  tone: SectorTone;
}

export interface ControlTowerModel {
  health: { healthy: number; total: number };
  activeRisk: { critical: number; total: number };
  nodes: SectorNode[];
  routes: SectorRoute[];
  traces: OverviewData["traces"];
  risks: OverviewData["risks"];
}

const positions = [
  [-2.7, 0.8, 0.2],
  [-1.6, 1.8, -0.3],
  [-0.7, 0.35, 0.7],
  [0.6, 1.7, -0.2],
  [1.9, 0.8, 0.4],
  [2.5, -0.5, -0.4],
  [1.1, -1.5, 0.5],
  [-0.4, -1.2, -0.1],
  [-1.9, -1.25, 0.35],
  [0.2, 2.35, 0.15],
  [2.85, 1.85, -0.1],
  [-2.8, 2.1, 0.25],
] as const;

export function buildControlTowerModel(
  data: OverviewData,
): ControlTowerModel {
  const openRisks = data.risks.filter((risk) => risk.status === "open");
  const criticalRiskTraceIds = new Set(
    openRisks
      .filter((risk) => risk.risk_level === "critical")
      .map((risk) => risk.trace_id),
  );

  const baseNodes = data.agents
    .slice(0, positions.length)
    .map((agent, index) => {
      const [x, y, z] = positions[index];
      const tone: SectorTone =
        agent.status === "active" ? "healthy" : "degraded";

      return {
        id: agent.id,
        label: agent.name,
        callsign: `AGT_${String(index + 1).padStart(2, "0")}`,
        tone,
        x,
        y,
        z,
      };
    });

  const nodesByName = new Map<string, SectorNode[]>();
  for (const node of baseNodes) {
    const matches = nodesByName.get(node.label);
    if (matches) {
      matches.push(node);
    } else {
      nodesByName.set(node.label, [node]);
    }
  }

  const criticalNodeIds = new Set<string>();
  for (const trace of data.traces) {
    if (!criticalRiskTraceIds.has(trace.traceId)) {
      continue;
    }

    const matches = nodesByName.get(trace.agentName);
    if (matches?.length === 1) {
      criticalNodeIds.add(matches[0].id);
    }
  }

  const nodes: SectorNode[] = baseNodes.map((node) => ({
    ...node,
    tone: criticalNodeIds.has(node.id) ? "critical" : node.tone,
  }));
  const hub = nodes[0];
  const routes = data.traces.flatMap<SectorRoute>((trace) => {
    const matches = nodesByName.get(trace.agentName);
    if (!hub || matches?.length !== 1) {
      return [];
    }

    const target = matches[0];

    const tone: SectorTone = criticalRiskTraceIds.has(trace.traceId)
      ? "critical"
      : trace.status === "failed" || trace.status === "timeout"
        ? "degraded"
        : "healthy";

    return [{ traceId: trace.traceId, fromId: hub.id, toId: target.id, tone }];
  });

  return {
    health: {
      healthy: nodes.filter((node) => node.tone === "healthy").length,
      total: nodes.length,
    },
    activeRisk: {
      critical: openRisks.filter((risk) => risk.risk_level === "critical").length,
      total: openRisks.length,
    },
    nodes,
    routes,
    traces: data.traces,
    risks: openRisks,
  };
}
