import { describe, expect, it } from "vitest";
import type { Span } from "../../lib/api/types";
import { buildSpanTree } from "./span-tree-model";

const span = (spanId: string, parentSpanId: string | null, sequence: number): Span => ({
  spanId,
  parentSpanId,
  spanType: spanId === "tool" ? "tool_call" : "llm_call",
  name: spanId,
  status: "success",
  sequence,
  durationMs: 1,
  startedAt: `2026-10-26T00:00:0${sequence}Z`,
});

describe("buildSpanTree", () => {
  it("orders every branch from its root by event sequence", () => {
    const roots = buildSpanTree([
      span("tool", "llm", 3),
      span("other-branch", "root", 2),
      span("root", null, 1),
      span("llm", "root", 2),
    ]);

    expect(roots.map((node) => node.span.spanId)).toEqual(["root"]);
    expect(roots[0].children.map((node) => node.span.spanId)).toEqual(["llm", "other-branch"]);
    expect(roots[0].children[0].children.map((node) => node.span.spanId)).toEqual(["tool"]);
  });

  it("keeps spans with missing parents visible as detached roots", () => {
    const roots = buildSpanTree([span("child", "missing", 2), span("root", null, 1)]);

    expect(roots.map((node) => node.span.spanId)).toEqual(["root", "child"]);
    expect(roots[1].detachedRoot).toBe(true);
  });

  it("breaks a malformed parent cycle instead of losing or infinitely nesting spans", () => {
    const roots = buildSpanTree([span("later", "earlier", 2), span("earlier", "later", 1)]);

    expect(roots).toHaveLength(1);
    expect(roots[0].span.spanId).toBe("earlier");
    expect(roots[0].children.map((node) => node.span.spanId)).toEqual(["later"]);
    expect(roots[0].detachedRoot).toBe(true);
    expect(roots[0].children[0].detachedRoot).toBe(false);
  });
});
