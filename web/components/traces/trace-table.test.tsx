import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TraceTable } from "./trace-table";
import type { TraceSummary } from "../../lib/api/types";

const trace: TraceSummary = {
  traceId: "trace_001",
  agentName: "IT Ops Agent",
  status: "success",
  riskLevel: "high",
  durationMs: 2480,
  totalTokens: 1580,
  estimatedCost: 0.0042,
  startedAt: "2026-08-17T10:00:00Z",
};

describe("TraceTable", () => {
  it("renders a trace summary row", () => {
    render(<TraceTable traces={[trace]} />);

    expect(screen.getByText("IT Ops Agent")).toBeInTheDocument();
    expect(screen.getByText("high")).toBeInTheDocument();
    expect(screen.getByText("1,580")).toBeInTheDocument();
  });

  it("renders a meaningful empty state", () => {
    render(<TraceTable traces={[]} />);

    expect(screen.getByText("No traces found")).toBeInTheDocument();
    expect(screen.getByText("Try changing the filters or wait for a new Agent event.")).toBeInTheDocument();
  });
});
