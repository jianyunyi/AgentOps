# Trace Investigation and Risk Disposition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give authorized operators an evidence-backed risk investigation and per-event disposition workflow inside Trace detail.

**Architecture:** The existing Trace detail route remains the composition boundary. A pure model filters and orders the existing risk-event result set; a focused component renders redacted evidence and action controls; the page owns permissions, request lifecycles, retry, and local mutation state.

**Tech Stack:** Next.js App Router, React 19, TypeScript, Vitest, React Testing Library, existing AgentOps REST clients and permission helper.

---

## File structure

- Create `web/components/traces/trace-risk-model.ts`: exact trace association and deterministic ordering.
- Create `web/components/traces/trace-risk-model.test.ts`: model behavior, malformed dates, and no-result behavior.
- Create `web/components/traces/trace-risk-evidence.tsx`: accessible loading, error, empty, evidence, and review states.
- Create `web/components/traces/trace-risk-evidence.test.tsx`: component state and action coverage.
- Create `web/app/dashboard/traces/[traceId]/page.test.tsx`: page permission, isolated request, and mutation orchestration.
- Modify `web/app/dashboard/traces/[traceId]/page.tsx`: session lookup, independent risk request, retry, and disposition state.
- Modify `web/app/globals.css`: responsive investigation layout and safe long-evidence wrapping.

### Task 1: Derive trace-specific risk evidence

**Files:**

- Create: `web/components/traces/trace-risk-model.ts`
- Test: `web/components/traces/trace-risk-model.test.ts`

- [ ] **Step 1: Write the failing model tests**

```ts
import { describe, expect, it } from "vitest";
import { risksForTrace } from "./trace-risk-model";

const risk = (id: string, trace_id: string, created_at: string) => ({
  id, tenant_id: "tenant-1", trace_id, span_id: "span-1", rule_code: "prompt_injection",
  risk_type: "prompt_injection", risk_level: "critical", detector: "rules", reason: id,
  evidence_redacted: "[redacted]", status: "open", created_at,
});

describe("risksForTrace", () => {
  it("keeps exact trace matches and orders newest valid records first", () => {
    expect(risksForTrace([risk("older", "trace-1", "2026-09-01T00:00:00Z"), risk("other", "trace-10", "2026-09-03T00:00:00Z"), risk("newer", "trace-1", "2026-09-02T00:00:00Z")], "trace-1").map(({ id }) => id)).toEqual(["newer", "older"]);
  });
  it("puts invalid dates last and resolves equal timestamps by id", () => {
    expect(risksForTrace([risk("z", "trace-1", "2026-09-02T00:00:00Z"), risk("a", "trace-1", "2026-09-02T00:00:00Z"), risk("invalid", "trace-1", "not-a-date")], "trace-1").map(({ id }) => id)).toEqual(["a", "z", "invalid"]);
  });
});
```

- [ ] **Step 2: Run it to verify it fails**

Run: `npm.cmd test -- components/traces/trace-risk-model.test.ts --reporter=dot`

Expected: FAIL because `trace-risk-model` is absent.

- [ ] **Step 3: Write the minimal model**

```ts
import type { RiskEvent } from "../../lib/api/types";

const timestamp = (value: string) => {
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? Number.NEGATIVE_INFINITY : parsed;
};

export const risksForTrace = (risks: RiskEvent[], traceId: string) => (
  risks.filter((risk) => risk.trace_id === traceId).sort((left, right) => (
    timestamp(right.created_at) - timestamp(left.created_at) || left.id.localeCompare(right.id)
  ))
);
```

- [ ] **Step 4: Verify the model passes**

Run: `npm.cmd test -- components/traces/trace-risk-model.test.ts --reporter=dot`

Expected: PASS, 2 tests.

- [ ] **Step 5: Commit**

Run: `git add web/components/traces/trace-risk-model.ts web/components/traces/trace-risk-model.test.ts && git commit -m "feat: derive trace risk evidence"`

### Task 2: Render bounded risk evidence and review controls

**Files:**

- Create: `web/components/traces/trace-risk-evidence.tsx`
- Test: `web/components/traces/trace-risk-evidence.test.tsx`

- [ ] **Step 1: Write failing component tests**

```tsx
it("renders server-redacted evidence and dispatches a disposition", async () => {
  const onReview = vi.fn().mockResolvedValue(undefined);
  render(<TraceRiskEvidence state={{ kind: "ready", risks: [risk] }} canReview onReview={onReview} />);
  expect(screen.getByText("[redacted]")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Resolve risk-1" }));
  expect(onReview).toHaveBeenCalledWith("risk-1", "resolved");
});

it("omits actions without permission and retries a local error", async () => {
  const onRetry = vi.fn();
  const { rerender } = render(<TraceRiskEvidence state={{ kind: "ready", risks: [risk] }} canReview={false} onReview={vi.fn()} />);
  expect(screen.queryByRole("button", { name: /Acknowledge/ })).not.toBeInTheDocument();
  rerender(<TraceRiskEvidence state={{ kind: "error", onRetry }} canReview={false} onReview={vi.fn()} />);
  await userEvent.click(screen.getByRole("button", { name: "Retry risk evidence" }));
  expect(onRetry).toHaveBeenCalledOnce();
});
```

- [ ] **Step 2: Verify the component test fails**

Run: `npm.cmd test -- components/traces/trace-risk-evidence.test.tsx --reporter=dot`

Expected: FAIL because `TraceRiskEvidence` is absent.

- [ ] **Step 3: Implement the state contract**

```tsx
export type TraceRiskState =
  | { kind: "loading" }
  | { kind: "error"; onRetry: () => void }
  | { kind: "ready"; risks: RiskEvent[] };

export function TraceRiskEvidence({ state, canReview, onReview }: {
  state: TraceRiskState; canReview: boolean; onReview: (id: string, status: "acknowledged" | "resolved") => Promise<void>;
}) {
  // Return an <aside aria-labelledby="trace-risk-heading"> with explicit state branches.
}
```

The ready state shows only risk level, rule, detector, reason, `evidence_redacted`, status, and creation time. It renders `StatusBadge domain="risk"`; its empty text is exactly `No associated risk events in loaded results.`. For an allowed reviewer, buttons are named `Acknowledge ${risk.id}` and `Resolve ${risk.id}`. Keep one local busy ID so both actions for only that event are disabled during its promise.

- [ ] **Step 4: Add loading, alert, empty, and pending-action tests; then run them**

Run: `npm.cmd test -- components/traces/trace-risk-evidence.test.tsx --reporter=dot`

Expected: PASS for loading status, error/retry, bounded empty state, permission, redaction, callback, and disabled-pending actions.

- [ ] **Step 5: Commit**

Run: `git add web/components/traces/trace-risk-evidence.tsx web/components/traces/trace-risk-evidence.test.tsx && git commit -m "feat: add trace risk evidence panel"`

### Task 3: Orchestrate permissions, isolated risk loading, and disposition

**Files:**

- Modify: `web/app/dashboard/traces/[traceId]/page.tsx`
- Create: `web/app/dashboard/traces/[traceId]/page.test.tsx`

- [ ] **Step 1: Write failing page orchestration tests**

Mock `getCurrentUser`, `getTrace`, `listRiskEvents`, `reviewRiskEvent`, and `useParams`. Add these cases:

```tsx
it("does not request or render risk evidence without risk:read", async () => {
  getCurrentUser.mockResolvedValue({ ...user, permissions: [] }); getTrace.mockResolvedValue(trace);
  render(<TraceDetailPage />);
  expect(await screen.findByRole("heading", { name: /Trace trace-1/ })).toBeInTheDocument();
  expect(listRiskEvents).not.toHaveBeenCalled();
  expect(screen.queryByRole("region", { name: "Risk evidence" })).not.toBeInTheDocument();
});

it("keeps timeline visible after optional-risk failure and retries only risks", async () => {
  getCurrentUser.mockResolvedValue({ ...user, permissions: ["risk:read"] }); getTrace.mockResolvedValue(trace);
  listRiskEvents.mockRejectedValueOnce(new Error("secret")).mockResolvedValueOnce({ data: [], pagination });
  render(<TraceDetailPage />);
  expect(await screen.findByText("Execution timeline")).toBeInTheDocument();
  expect(await screen.findByRole("alert")).toHaveTextContent("Risk evidence is temporarily unavailable.");
  await userEvent.click(screen.getByRole("button", { name: "Retry risk evidence" }));
  expect(getTrace).toHaveBeenCalledTimes(1); expect(listRiskEvents).toHaveBeenCalledTimes(2);
});
```

- [ ] **Step 2: Verify route tests fail**

Run: `npm.cmd test -- "app/dashboard/traces/[traceId]/page.test.tsx" --reporter=dot`

Expected: FAIL because the route has no session/risk orchestration.

- [ ] **Step 3: Implement page orchestration**

Import `getCurrentUser`, `listRiskEvents`, `reviewRiskEvent`, `hasPermission`, `TraceRiskEvidence`, `TraceRiskState`, and `risksForTrace`. Maintain `canReadRisks`, `canReviewRisks`, `riskState`, and a retry version. In a route-keyed effect, load the user; if `risk:read`, request `listRiskEvents` and set `{ kind: "ready", risks: risksForTrace(page.data, params.traceId) }`. Use a cancellation boolean so stale route/retry responses cannot write state. On risk failure, set the bounded error `Risk evidence is temporarily unavailable.` through the component state; retry increments only the risk request version. Do not alter the existing trace request and its page-level failure behavior.

Define `reviewRisk` to await `reviewRiskEvent(id, status)` and replace only the matching ready-state risk. Re-throw `new Error("Unable to update risk event")` on failure so the component never exposes server text.

- [ ] **Step 4: Add and pass review tests**

Add tests for read-only `risk:read`, successful acknowledgement of exactly one event, and rejected review preserving `open` with `Unable to update risk event` visible. Run:

Run: `npm.cmd test -- "app/dashboard/traces/[traceId]/page.test.tsx" --reporter=dot`

Expected: PASS with permission, retry isolation, success, and failure coverage.

- [ ] **Step 5: Commit**

Run: `git add "web/app/dashboard/traces/[traceId]/page.tsx" "web/app/dashboard/traces/[traceId]/page.test.tsx" && git commit -m "feat: add trace risk investigation workflow"`

### Task 4: Make the investigation layout responsive and verify it

**Files:**

- Modify: `web/app/globals.css`
- Modify: `web/components/traces/trace-risk-evidence.tsx`
- Modify: `web/components/traces/trace-risk-evidence.test.tsx`

- [ ] **Step 1: Write a failing class contract test**

Mount ready evidence with a long unbroken `evidence_redacted` value and assert the rendered evidence has `trace-risk-evidence__redacted`.

- [ ] **Step 2: Verify it fails**

Run: `npm.cmd test -- components/traces/trace-risk-evidence.test.tsx --reporter=dot`

Expected: FAIL because the redacted node has no wrapping hook class.

- [ ] **Step 3: Implement the class and CSS contract**

Add `className="trace-risk-evidence__redacted"` to the evidence node and append:

```css
.trace-investigation { display:grid; grid-template-columns:minmax(0,1.45fr) minmax(280px,.55fr); gap:18px; align-items:start; }
.trace-timeline, .trace-risk-evidence { min-width:0; padding:24px; }
.trace-risk-evidence { border:1px solid var(--line); border-radius:var(--radius-md); background:var(--surface-raised); }
.trace-risk-evidence__redacted { min-width:0; overflow-wrap:anywhere; white-space:pre-wrap; }
.trace-risk-evidence__item { padding:14px 0; border-top:1px solid var(--line); }
@media (max-width:900px) { .trace-investigation { grid-template-columns:1fr; } }
```

Wrap the existing timeline panel and evidence component in `.trace-investigation`; keep `SpanTree`, the heading, and Trace summary unchanged.

- [ ] **Step 4: Run full validation**

Run: `npm.cmd test -- --reporter=dot`

Expected: all frontend tests PASS.

Run: `npx.cmd --no-install tsc --noEmit`

Expected: exit code 0.

Run: `npm.cmd run build`

Expected: production build succeeds.

Run: `git diff --check`

Expected: no whitespace errors.

- [ ] **Step 5: Commit**

Run: `git add web/app/globals.css web/components/traces/trace-risk-evidence.tsx web/components/traces/trace-risk-evidence.test.tsx && git commit -m "style: polish trace investigation layout"`
