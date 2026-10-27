import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getCurrentUser, getTrace, listRiskEvents, reviewRiskEvent } = vi.hoisted(() => ({
  getCurrentUser: vi.fn(), getTrace: vi.fn(), listRiskEvents: vi.fn(), reviewRiskEvent: vi.fn(),
}));

vi.mock("../../../../lib/api/auth", () => ({ getCurrentUser }));
vi.mock("../../../../lib/api/client", () => ({ APIError: class APIError extends Error {}, getTrace }));
vi.mock("../../../../lib/api/governance", () => ({ listRiskEvents, reviewRiskEvent }));
vi.mock("next/navigation", () => ({ useParams: () => ({ traceId: "trace-1" }) }));

import TraceDetailPage from "./page";

const user = { user_id: "user-1", tenant_id: "tenant-1", role: "owner", permissions: [] };
const trace = { traceId: "trace-1", tenantId: "tenant-1", agentId: "agent-1", agentName: "Support Agent", status: "success", riskLevel: "high", durationMs: 120, totalTokens: 42, estimatedCost: 0.01, startedAt: "2026-09-07T10:00:00Z", spans: [] };
const risk = { id: "risk-1", tenant_id: "tenant-1", trace_id: "trace-1", span_id: "span-1", rule_code: "prompt_injection", risk_type: "prompt_injection", risk_level: "high", detector: "rules", reason: "Detected instruction override", evidence_redacted: "[redacted]", status: "open", created_at: "2026-09-07T10:00:00Z" };
const page = { data: [risk], pagination: { page: 1, page_size: 20, total: 1 } };

describe("TraceDetailPage", () => {
  beforeEach(() => {
    getCurrentUser.mockResolvedValue(user);
    getTrace.mockResolvedValue(trace);
    listRiskEvents.mockResolvedValue(page);
    reviewRiskEvent.mockResolvedValue(undefined);
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("does not request or render risk evidence without risk:read", async () => {
    render(<TraceDetailPage />);
    expect(await screen.findByRole("heading", { name: "Trace trace-1" })).toBeInTheDocument();
    expect(listRiskEvents).not.toHaveBeenCalled();
    expect(screen.queryByRole("complementary", { name: "Risk evidence" })).not.toBeInTheDocument();
  });

  it("opens the full root-first execution chain from the side icon", async () => {
    getTrace.mockResolvedValue({ ...trace, spans: [
      { spanId: "tool", parentSpanId: "llm", spanType: "tool_call", name: "tool_call", status: "success", sequence: 3, inputSnapshot: { function: "secret_function_name" }, durationMs: 12, startedAt: "2026-10-26T10:00:03Z" },
      { spanId: "llm", parentSpanId: "root", spanType: "llm_call", name: "llm_call", status: "success", sequence: 2, durationMs: 50, startedAt: "2026-10-26T10:00:02Z" },
      { spanId: "root", parentSpanId: null, spanType: "trace_start", name: "trace_start", status: "success", sequence: 1, durationMs: 0, startedAt: "2026-10-26T10:00:01Z" },
    ] });

    render(<TraceDetailPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Open full execution trace" }));

    const dialog = await screen.findByRole("dialog", { name: "Full execution chain" });
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Close" })).toHaveFocus();
    expect(within(dialog).getByText("trace_start → llm_call → tool_call")).toBeInTheDocument();
    expect(screen.queryByText(/secret_function_name/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Input payload/)).not.toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Full execution chain" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open full execution trace" })).toHaveFocus();
  });

  it("keeps the timeline visible when risk evidence fails and retries only that request", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: ["risk:read"] });
    listRiskEvents.mockRejectedValueOnce(new Error("server secret")).mockResolvedValueOnce({ data: [], pagination: page.pagination });
    render(<TraceDetailPage />);
    expect(await screen.findByText("Execution timeline")).toBeInTheDocument();
    expect(await screen.findByRole("alert")).toHaveTextContent("Risk evidence is temporarily unavailable.");
    await userEvent.click(screen.getByRole("button", { name: "Retry risk evidence" }));
    expect(getTrace).toHaveBeenCalledTimes(1);
    expect(listRiskEvents).toHaveBeenCalledTimes(2);
  });

  it("updates only a permitted reviewed risk", async () => {
    getCurrentUser.mockResolvedValue({ ...user, permissions: ["risk:read", "risk:review"] });
    render(<TraceDetailPage />);
    await userEvent.click(await screen.findByRole("button", { name: "Acknowledge risk-1" }));
    expect(reviewRiskEvent).toHaveBeenCalledWith("risk-1", "acknowledged");
    expect(screen.getByText("acknowledged")).toBeInTheDocument();
  });
});
