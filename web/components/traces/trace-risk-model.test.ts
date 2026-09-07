import { describe, expect, it } from "vitest";
import type { RiskEvent } from "../../lib/api/types";
import { risksForTrace } from "./trace-risk-model";

const risk = (id: string, trace_id: string, created_at: string): RiskEvent => ({
  id,
  tenant_id: "tenant-1",
  trace_id,
  span_id: "span-1",
  rule_code: "prompt-injection",
  risk_type: "prompt_injection",
  risk_level: "high",
  detector: "rules",
  reason: "Matched rule",
  evidence_redacted: "[redacted]",
  status: "open",
  created_at,
});

describe("risksForTrace", () => {
  it("matches only the exact trace id", () => {
    expect(
      risksForTrace(
        [
          risk("exact", "trace-123", "2026-09-07T10:00:00Z"),
          risk("prefix", "trace-1234", "2026-09-07T10:01:00Z"),
          risk("suffix", "before-trace-123", "2026-09-07T10:02:00Z"),
        ],
        "trace-123",
      ).map(({ id }) => id),
    ).toEqual(["exact"]);
  });

  it("returns no risks when no event exactly matches the requested trace id", () => {
    expect(
      risksForTrace(
        [
          risk("close", "trace-10", "2026-09-07T10:00:00Z"),
          risk("other", "trace-2", "2026-09-07T10:01:00Z"),
        ],
        "trace-1",
      ),
    ).toEqual([]);
  });

  it("sorts valid timestamps newest first", () => {
    expect(
      risksForTrace(
        [
          risk("old", "trace-1", "2026-09-07T08:00:00Z"),
          risk("new", "trace-1", "2026-09-07T10:00:00Z"),
          risk("middle", "trace-1", "2026-09-07T09:00:00Z"),
        ],
        "trace-1",
      ).map(({ id }) => id),
    ).toEqual(["new", "middle", "old"]);
  });

  it("places malformed timestamps after valid timestamps", () => {
    expect(
      risksForTrace(
        [
          risk("malformed", "trace-1", "not-a-timestamp"),
          risk("valid", "trace-1", "2026-09-07T10:00:00Z"),
        ],
        "trace-1",
      ).map(({ id }) => id),
    ).toEqual(["valid", "malformed"]);
  });

  it("uses lexical id order for equal timestamps", () => {
    expect(
      risksForTrace(
        [
          risk("zeta", "trace-1", "2026-09-07T10:00:00Z"),
          risk("alpha", "trace-1", "2026-09-07T10:00:00Z"),
        ],
        "trace-1",
      ).map(({ id }) => id),
    ).toEqual(["alpha", "zeta"]);
  });

  it("leaves the caller input order unchanged", () => {
    const risks = [
      risk("old", "trace-1", "2026-09-07T08:00:00Z"),
      risk("new", "trace-1", "2026-09-07T10:00:00Z"),
    ];

    risksForTrace(risks, "trace-1");

    expect(risks.map(({ id }) => id)).toEqual(["old", "new"]);
  });
});
