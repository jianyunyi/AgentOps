import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getCurrentUser } = vi.hoisted(() => ({ getCurrentUser: vi.fn() }));
const { getOverviewData } = vi.hoisted(() => ({ getOverviewData: vi.fn() }));

vi.mock("../../lib/api/auth", () => ({ getCurrentUser }));
vi.mock("../../lib/api/overview", () => ({ getOverviewData }));

import { OverviewSignals } from "./overview-signals";

const user = { user_id: "user-1", tenant_id: "tenant-1", role: "owner", permissions: ["agent:read", "risk:read"] };
const agent = { id: "agent-1", tenantId: "tenant-1", name: "Support Agent", description: "", environment: "production", status: "active" };
const trace = { traceId: "trace-1", agentName: "Support Agent", status: "success", riskLevel: "none", durationMs: 120, totalTokens: 80, estimatedCost: 0.01, startedAt: "2026-09-03T00:00:00Z" };
const risk = { id: "risk-1", tenant_id: "tenant-1", trace_id: "trace-1", span_id: "span-1", rule_code: "prompt_injection", risk_type: "prompt_injection", risk_level: "high", detector: "rules", reason: "Detected instruction override", evidence_redacted: "[redacted]", status: "open", created_at: "2026-09-03T00:00:00Z" };

describe("OverviewSignals", () => {
  beforeEach(() => {
    getCurrentUser.mockResolvedValue(user);
    getOverviewData.mockResolvedValue({ agents: [agent], traces: [trace], risks: [risk] });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the overview signal summary from API data", async () => {
    render(<OverviewSignals />);

    expect(await screen.findByText("1 active Agent")).toBeInTheDocument();
    expect(screen.getByText("1 open risk")).toBeInTheDocument();
    expect(screen.getByText("Support Agent")).toBeInTheDocument();
  });

  it("does not invent metrics when the API is empty", async () => {
    getOverviewData.mockResolvedValue({ agents: [], traces: [], risks: [] });
    render(<OverviewSignals />);

    expect(await screen.findByText("No live signals yet")).toBeInTheDocument();
    expect(screen.queryByText(/active Agent/)).not.toBeInTheDocument();
  });

  it("shows a forbidden state when the user cannot read Agent data", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: [] });
    render(<OverviewSignals />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Overview access is restricted");
    expect(getOverviewData).not.toHaveBeenCalled();
  });
});
