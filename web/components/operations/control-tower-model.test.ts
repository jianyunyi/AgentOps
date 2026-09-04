import { describe, expect, it } from "vitest";
import type {
  Agent,
  RiskEvent,
  TraceStatus,
  TraceSummary,
} from "../../lib/api/types";
import { buildControlTowerModel } from "./control-tower-model";

const agent = (id: string, name: string, status = "active"): Agent => ({
  id,
  tenantId: "tenant-1",
  name,
  description: "",
  environment: "production",
  status,
});

const trace = (
  traceId: string,
  agentName: string,
  status: TraceStatus = "success",
): TraceSummary => ({
  traceId,
  agentName,
  status,
  riskLevel: "none",
  durationMs: 100,
  totalTokens: 10,
  estimatedCost: 0.01,
  startedAt: "2026-09-04T06:00:00Z",
});

const criticalRisk = (
  traceId: string,
  status = "open",
): RiskEvent => ({
  id: `risk-${traceId}`,
  tenant_id: "tenant-1",
  trace_id: traceId,
  span_id: "span-1",
  rule_code: "egress",
  risk_type: "unauthorized_egress",
  risk_level: "critical",
  detector: "rules",
  reason: "Unauthorized egress",
  evidence_redacted: "[redacted]",
  status,
  created_at: "2026-09-04T06:00:01Z",
});

const expectedPositions = [
  { x: -2.7, y: 0.8, z: 0.2 },
  { x: -1.6, y: 1.8, z: -0.3 },
  { x: -0.7, y: 0.35, z: 0.7 },
  { x: 0.6, y: 1.7, z: -0.2 },
  { x: 1.9, y: 0.8, z: 0.4 },
  { x: 2.5, y: -0.5, z: -0.4 },
  { x: 1.1, y: -1.5, z: 0.5 },
  { x: -0.4, y: -1.2, z: -0.1 },
  { x: -1.9, y: -1.25, z: 0.35 },
  { x: 0.2, y: 2.35, z: 0.15 },
  { x: 2.85, y: 1.85, z: -0.1 },
  { x: -2.8, y: 2.1, z: 0.25 },
] as const;

describe("buildControlTowerModel", () => {
  it("derives fleet health and critical risk from live records", () => {
    const model = buildControlTowerModel({
      agents: [
        agent("a1", "orchestrator"),
        agent("a2", "egress-agent", "disabled"),
      ],
      traces: [trace("trc_8421", "egress-agent", "failed")],
      risks: [criticalRisk("trc_8421")],
    });

    expect(model.health).toEqual({ healthy: 1, total: 2 });
    expect(model.activeRisk).toEqual({ critical: 1, total: 1 });
    expect(model.nodes.map((node) => node.id)).toEqual(["a1", "a2"]);
    expect(model.routes[0]).toMatchObject({
      traceId: "trc_8421",
      tone: "critical",
    });
  });

  it("returns no invented nodes or routes for an empty workspace", () => {
    const model = buildControlTowerModel({ agents: [], traces: [], risks: [] });

    expect(model.nodes).toEqual([]);
    expect(model.routes).toEqual([]);
    expect(model.health).toEqual({ healthy: 0, total: 0 });
  });

  it("ignores closed critical risks", () => {
    const model = buildControlTowerModel({
      agents: [agent("hub", "orchestrator"), agent("target", "worker")],
      traces: [trace("trc_closed", "worker")],
      risks: [criticalRisk("trc_closed", "closed")],
    });

    expect(model.activeRisk).toEqual({ critical: 0, total: 0 });
    expect(model.nodes.map((node) => node.tone)).toEqual([
      "healthy",
      "healthy",
    ]);
  });

  it("gives critical risk priority over a non-active agent status", () => {
    const model = buildControlTowerModel({
      agents: [
        agent("hub", "orchestrator"),
        agent("target", "worker", "disabled"),
      ],
      traces: [trace("trc_critical", "worker")],
      risks: [criticalRisk("trc_critical")],
    });

    expect(model.nodes.find((node) => node.id === "target")?.tone).toBe(
      "critical",
    );
  });

  it("marks successful routes as healthy", () => {
    const model = buildControlTowerModel({
      agents: [agent("hub", "orchestrator"), agent("target", "worker")],
      traces: [trace("trc_success", "worker")],
      risks: [],
    });

    expect(model.routes[0]?.tone).toBe("healthy");
  });

  it("marks failed routes as degraded", () => {
    const model = buildControlTowerModel({
      agents: [agent("hub", "orchestrator"), agent("target", "worker")],
      traces: [trace("trc_failed", "worker", "failed")],
      risks: [],
    });

    expect(model.routes[0]?.tone).toBe("degraded");
  });

  it("marks timeout routes as degraded", () => {
    const model = buildControlTowerModel({
      agents: [agent("hub", "orchestrator"), agent("target", "worker")],
      traces: [trace("trc_timeout", "worker", "timeout")],
      risks: [],
    });

    expect(model.routes).toEqual([
      {
        traceId: "trc_timeout",
        fromId: "hub",
        toId: "target",
        tone: "degraded",
      },
    ]);
  });

  it("does not attribute traces to duplicate agent names", () => {
    const model = buildControlTowerModel({
      agents: [agent("a1", "worker"), agent("a2", "worker", "disabled")],
      traces: [trace("trc_duplicate", "worker")],
      risks: [criticalRisk("trc_duplicate")],
    });

    expect.soft(model.routes).toEqual([]);
    expect.soft(model.nodes.map((node) => node.tone)).toEqual([
      "healthy",
      "degraded",
    ]);
  });

  it("rejects traces whose target is not a displayed agent", () => {
    const model = buildControlTowerModel({
      agents: [agent("hub", "orchestrator")],
      traces: [trace("trc_missing", "missing-agent")],
      risks: [criticalRisk("trc_missing")],
    });

    expect(model.routes).toEqual([]);
    expect(model.nodes[0]?.tone).toBe("healthy");
  });

  it("uses at most 12 deterministic nodes", () => {
    const agents = Array.from({ length: 13 }, (_, index) =>
      agent(`a${index + 1}`, `agent-${index + 1}`),
    );
    const firstModel = buildControlTowerModel({ agents, traces: [], risks: [] });
    const secondModel = buildControlTowerModel({ agents, traces: [], risks: [] });

    expect(firstModel.nodes).toHaveLength(12);
    expect(firstModel.nodes.map((node) => node.id)).toEqual(
      agents.slice(0, 12).map(({ id }) => id),
    );
    expect(
      firstModel.nodes.map(({ callsign, x, y, z }) => ({
        callsign,
        x,
        y,
        z,
      })),
    ).toEqual(
      expectedPositions.map((position, index) => ({
        callsign: `AGT_${String(index + 1).padStart(2, "0")}`,
        ...position,
      })),
    );
    expect(secondModel.nodes).toEqual(firstModel.nodes);
  });
});
