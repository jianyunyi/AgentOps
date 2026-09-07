import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getCurrentUser } = vi.hoisted(() => ({ getCurrentUser: vi.fn() }));
const { getOverviewData } = vi.hoisted(() => ({ getOverviewData: vi.fn() }));
const { listPolicies } = vi.hoisted(() => ({ listPolicies: vi.fn() }));

vi.mock("../../lib/api/auth", () => ({ getCurrentUser }));
vi.mock("../../lib/api/overview", () => ({ getOverviewData }));
vi.mock("../../lib/api/policies", () => ({ listPolicies }));
vi.mock("next/link", () => ({ default: ({ href, children }: { href: string; children: React.ReactNode }) => <a href={href}>{children}</a> }));
vi.mock("./sector-topology", () => ({
  SectorTopology: ({ nodes, selectedId, onSelect }: { nodes: Array<{ id: string; label: string }>; selectedId?: string; onSelect?: (id: string) => void }) => (
    <div data-testid="topology" data-selected={selectedId}>
      {nodes.map((node) => <button key={node.id} type="button" onClick={() => onSelect?.(node.id)}>{node.label}</button>)}
    </div>
  ),
}));

import { ControlTower } from "./control-tower";

const user = { user_id: "user-1", tenant_id: "tenant-1", role: "owner", permissions: ["agent:read", "risk:read"] };
const agent = { id: "agent-1", tenantId: "tenant-1", name: "Support Agent", description: "", environment: "production", status: "active" };
const secondAgent = { ...agent, id: "agent-2", name: "Review Agent", status: "paused" };
const trace = { traceId: "trace/a", agentName: "Support Agent", status: "success", riskLevel: "high", durationMs: 120, totalTokens: 80, estimatedCost: 0.01, startedAt: "2026-09-03T00:00:00Z" };
const risk = { id: "risk-1", tenant_id: "tenant-1", trace_id: "trace/a", span_id: "span-1", rule_code: "prompt_injection", risk_type: "prompt_injection", risk_level: "critical", detector: "rules", reason: "Detected instruction override", evidence_redacted: "[redacted]", status: "open", created_at: "2026-09-03T00:00:00Z" };
const policy = { id: "policy-1", tenant_id: "tenant-1", name: "Production guardrails", version: 3, enabled: true, rules_enabled: true, llm_enabled: false, max_input_bytes: 1024, created_by: "user-1", created_at: "2026-09-03T00:00:00Z" };

describe("ControlTower", () => {
  beforeEach(() => {
    getCurrentUser.mockResolvedValue(user);
    getOverviewData.mockResolvedValue({ agents: [agent, secondAgent], traces: [trace], risks: [risk] });
    listPolicies.mockResolvedValue([]);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders live KPIs and links from loaded data", async () => {
    render(<ControlTower />);

    expect(await screen.findByRole("region", { name: "Agent operations control tower" })).toBeInTheDocument();
    expect(screen.getByText("1 / 2 active")).toBeInTheDocument();
    expect(screen.getByText("1 critical")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "trace/a" })).toHaveAttribute("href", "/dashboard/traces/trace%2Fa");
    expect(screen.getByRole("link", { name: "Detected instruction override" })).toHaveAttribute("href", "/dashboard/risk");
  });

  it("does not fetch overview data when access is forbidden", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: ["agent:read"] });
    render(<ControlTower />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Overview access is restricted");
    expect(getOverviewData).not.toHaveBeenCalled();
  });

  it("does not invent KPIs for an empty overview", async () => {
    getOverviewData.mockResolvedValue({ agents: [], traces: [], risks: [] });
    render(<ControlTower />);

    expect(await screen.findByText("No live signals yet")).toBeInTheDocument();
    expect(screen.queryByText(/Registry status/)).not.toBeInTheDocument();
  });

  it("uses bounded error text and retries", async () => {
    getOverviewData.mockRejectedValueOnce(new Error("internal database topology"));
    render(<ControlTower />);

    expect(await screen.findByRole("alert")).toHaveTextContent("The console could not load this workspace snapshot.");
    expect(screen.queryByText("internal database topology")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("region", { name: "Agent operations control tower" })).toBeInTheDocument();
    expect(getOverviewData).toHaveBeenCalledTimes(2);
  });

  it("defaults selection to the first node and updates it from topology", async () => {
    render(<ControlTower />);

    expect(await screen.findByTestId("topology")).toHaveAttribute("data-selected", "agent-1");
    await userEvent.click(screen.getByRole("button", { name: "Review Agent" }));
    expect(screen.getByTestId("topology")).toHaveAttribute("data-selected", "agent-2");
    expect(screen.getByText("Review Agent", { selector: "dd" })).toBeInTheDocument();
  });

  it("renders policy posture after loading policies for a permitted user", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: [...user.permissions, "policy:read"] });
    listPolicies.mockResolvedValue([policy]);
    render(<ControlTower />);

    expect(await screen.findByRole("heading", { name: "Policy posture" })).toBeInTheDocument();
    expect(screen.getByText("Production guardrails · v3")).toBeInTheDocument();
    expect(listPolicies).toHaveBeenCalledTimes(1);
  });

  it("does not request or render policy posture without policy permission", async () => {
    render(<ControlTower />);

    await screen.findByRole("region", { name: "Agent operations control tower" });
    expect(listPolicies).not.toHaveBeenCalled();
    expect(screen.queryByRole("heading", { name: "Policy posture" })).not.toBeInTheDocument();
  });

  it("retries only the optional policy request after a policy posture error", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: [...user.permissions, "policy:read"] });
    listPolicies.mockRejectedValueOnce(new Error("provider secret leaked")).mockResolvedValueOnce([policy]);
    render(<ControlTower />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Policy posture is temporarily unavailable.");
    expect(screen.queryByText("provider secret leaked")).not.toBeInTheDocument();
    expect(getOverviewData).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByRole("button", { name: "Retry policy posture" }));
    expect(await screen.findByText("Production guardrails · v3")).toBeInTheDocument();
    expect(listPolicies).toHaveBeenCalledTimes(2);
    expect(getOverviewData).toHaveBeenCalledTimes(1);
  });
});
