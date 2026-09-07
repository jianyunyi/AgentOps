import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { RiskEvent } from "../../lib/api/types";
import { TraceRiskEvidence } from "./trace-risk-evidence";

const risk = (id = "risk-1"): RiskEvent => ({
  id,
  tenant_id: "tenant-1",
  trace_id: "trace-1",
  span_id: "span-1",
  rule_code: "prompt_injection",
  risk_type: "prompt_injection",
  risk_level: "high",
  detector: "rules",
  reason: "Detected instruction override",
  evidence_redacted: "[REDACTED: external instruction]",
  status: "open",
  created_at: "2026-09-07T10:00:00Z",
});

const ready = (risks: RiskEvent[] = [risk()]) => ({ kind: "ready" as const, risks });

describe("TraceRiskEvidence", () => {
  afterEach(cleanup);

  it("announces loading in its named evidence region", () => {
    render(<TraceRiskEvidence state={{ kind: "loading" }} canReview={false} onReview={vi.fn()} />);

    expect(screen.getByRole("complementary", { name: "Risk evidence" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("bounds a failed request and retries", async () => {
    const onRetry = vi.fn();
    render(<TraceRiskEvidence state={{ kind: "error", onRetry }} canReview={false} onReview={vi.fn()} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Risk evidence is temporarily unavailable.");
    await userEvent.click(screen.getByRole("button", { name: "Retry risk evidence" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("states when loaded results contain no associated risks", () => {
    render(<TraceRiskEvidence state={ready([])} canReview={false} onReview={vi.fn()} />);

    expect(screen.getByText("No associated risk events in loaded results.")).toBeInTheDocument();
  });

  it("renders only supplied redacted evidence and risk metadata", () => {
    render(<TraceRiskEvidence state={ready()} canReview={false} onReview={vi.fn()} />);

    expect(screen.getByText("[REDACTED: external instruction]")).toBeInTheDocument();
    expect(screen.getByText("prompt_injection")).toBeInTheDocument();
    expect(screen.getByText("rules")).toBeInTheDocument();
    expect(screen.getByText("Detected instruction override")).toBeInTheDocument();
    expect(screen.getByText("high")).toHaveClass("status-badge--danger");
    expect(screen.getByText("open")).toHaveClass("status-badge--danger");
    expect(screen.getByText("2026-09-07T10:00:00.000Z")).toBeInTheDocument();
  });

  it("marks long redacted evidence for safe wrapping", () => {
    render(<TraceRiskEvidence state={ready([{ ...risk(), evidence_redacted: "evidence-with-a-very-long-unbroken-value" }])} canReview={false} onReview={vi.fn()} />);

    expect(screen.getByText("evidence-with-a-very-long-unbroken-value")).toHaveClass("trace-risk-evidence__redacted");
  });

  it("does not expose review controls without permission", () => {
    render(<TraceRiskEvidence state={ready()} canReview={false} onReview={vi.fn()} />);

    expect(screen.queryByRole("button", { name: /Acknowledge/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Resolve/ })).not.toBeInTheDocument();
  });

  it("submits acknowledge and resolve operations", async () => {
    const onReview = vi.fn().mockResolvedValue(undefined);
    render(<TraceRiskEvidence state={ready([risk("risk-1"), risk("risk-2")])} canReview onReview={onReview} />);

    await userEvent.click(screen.getByRole("button", { name: "Acknowledge risk-1" }));
    await userEvent.click(screen.getByRole("button", { name: "Resolve risk-2" }));

    expect(onReview).toHaveBeenNthCalledWith(1, "risk-1", "acknowledged");
    expect(onReview).toHaveBeenNthCalledWith(2, "risk-2", "resolved");
  });

  it("disables only the pending event controls", async () => {
    let resolveReview!: () => void;
    const onReview = vi.fn(() => new Promise<void>((resolve) => { resolveReview = resolve; }));
    render(<TraceRiskEvidence state={ready([risk("risk-1"), risk("risk-2")])} canReview onReview={onReview} />);

    await userEvent.click(screen.getByRole("button", { name: "Acknowledge risk-1" }));

    expect(screen.getByRole("button", { name: "Acknowledge risk-1" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Resolve risk-1" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Acknowledge risk-2" })).not.toBeDisabled();
    resolveReview();
  });

  it("announces a rejected review without removing the risk", async () => {
    const onReview = vi.fn().mockRejectedValue(new Error("server details must not render"));
    render(<TraceRiskEvidence state={ready()} canReview onReview={onReview} />);

    await userEvent.click(screen.getByRole("button", { name: "Resolve risk-1" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/^Unable to update risk event$/);
    expect(screen.getByText("[REDACTED: external instruction]")).toBeInTheDocument();
    const riskEvent = within(screen.getByRole("complementary")).getByText("risk-1").closest("article");
    expect(riskEvent).not.toBeNull();
    expect(within(riskEvent as HTMLElement).getByText("open")).toBeInTheDocument();
  });
});
