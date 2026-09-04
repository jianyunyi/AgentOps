import { describe, expect, it } from "vitest";
import { buildControlTowerModel } from "./control-tower-model";

const agent = (id: string, name: string, status = "active") => ({
  id,
  tenantId: "tenant-1",
  name,
  description: "",
  environment: "production",
  status,
});

describe("buildControlTowerModel", () => {
  it("derives fleet health and critical risk from live records", () => {
    const model = buildControlTowerModel({
      agents: [
        agent("a1", "orchestrator"),
        agent("a2", "egress-agent", "disabled"),
      ],
      traces: [
        {
          traceId: "trc_8421",
          agentName: "egress-agent",
          status: "failed",
          riskLevel: "critical",
          durationMs: 842,
          totalTokens: 90,
          estimatedCost: 0.02,
          startedAt: "2026-09-04T06:00:00Z",
        },
      ],
      risks: [
        {
          id: "r1",
          tenant_id: "tenant-1",
          trace_id: "trc_8421",
          span_id: "s1",
          rule_code: "egress",
          risk_type: "unauthorized_egress",
          risk_level: "critical",
          detector: "rules",
          reason: "Unauthorized egress",
          evidence_redacted: "[redacted]",
          status: "open",
          created_at: "2026-09-04T06:00:01Z",
        },
      ],
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
});
