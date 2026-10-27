import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { Span } from "../../lib/api/types";
import { SpanTree } from "./span-tree";

const spans: Span[] = [
  {
    spanId: "tool-1", parentSpanId: "llm-1", spanType: "tool_call", name: "tool_call",
    status: "success", sequence: 3, inputSnapshot: { function: "secret_function_name" },
    durationMs: 18, startedAt: "2026-10-26T10:00:03Z",
  },
  {
    spanId: "llm-1", parentSpanId: "root-1", spanType: "llm_call", name: "llm_call",
    status: "success", sequence: 2, durationMs: 80, startedAt: "2026-10-26T10:00:02Z",
  },
  {
    spanId: "root-1", parentSpanId: null, spanType: "trace_start", name: "trace_start",
    status: "success", sequence: 1, durationMs: 0, startedAt: "2026-10-26T10:00:01Z",
  },
];

afterEach(() => cleanup());

describe("SpanTree", () => {
  it("shows the source-to-tool path without rendering raw span snapshots", () => {
    render(<SpanTree spans={spans} />);

    expect(screen.getByText("trace_start → llm_call → tool_call")).toBeInTheDocument();
    expect(screen.getByText("Span tool-1 · parent llm-1")).toBeInTheDocument();
    expect(screen.queryByText(/secret_function_name/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Input payload/)).not.toBeInTheDocument();
  });

  it("renders an explicit empty state when no spans were recorded", () => {
    render(<SpanTree spans={[]} />);
    expect(screen.getByText("No spans recorded for this trace.")).toBeInTheDocument();
  });
});
