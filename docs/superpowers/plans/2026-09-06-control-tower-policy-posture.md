# Control Tower Policy Posture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a permission-aware, real-data policy posture panel to the control tower.

**Architecture:** `buildPolicyPosture` projects existing tenant-scoped `Policy[]` data. `PolicyPosture` presents state without networking. `ControlTower` performs an optional policy request after its current Agent/Risk gate and retries it independently.

**Tech Stack:** Next.js 16, React 19, TypeScript, Vitest, Testing Library, `GET /api/v1/policies`.

---

### Task 1: Add a deterministic policy model

**Files:**
- Create: `web/components/operations/policy-posture-model.ts`
- Create: `web/components/operations/policy-posture-model.test.ts`

- [ ] **Step 1: Write the failing tests**

```ts
const policy = (id: string, version: number, enabled = true, created_at = "2026-09-06T10:00:00Z") => ({ id, tenant_id: "tenant-1", name: `Policy ${id}`, version, enabled, rules_enabled: true, llm_enabled: false, max_input_bytes: 2048, created_by: "user-1", created_at });

it("counts enabled policies and selects the greatest enabled version", () => {
  expect(buildPolicyPosture([policy("old", 2), policy("current", 3), policy("disabled", 9, false)])).toMatchObject({ enabledCount: 2, active: { id: "current" } });
});

it("uses newest timestamp then lexical id for equal versions", () => {
  expect(buildPolicyPosture([policy("zeta", 3, true, "2026-09-05T00:00:00Z"), policy("alpha", 3, true, "2026-09-06T00:00:00Z")]).active?.id).toBe("alpha");
});

it("does not invent an active policy", () => {
  expect(buildPolicyPosture([])).toEqual({ enabledCount: 0, active: undefined });
  expect(buildPolicyPosture([policy("disabled", 1, false)])).toEqual({ enabledCount: 0, active: undefined });
});
```

- [ ] **Step 2: Verify RED**

Run: `npm.cmd test -- components/operations/policy-posture-model.test.ts`

Expected: the module does not yet exist.

- [ ] **Step 3: Implement the model**

```ts
import type { Policy } from "../../lib/api/types";

export type PolicyPosture = { enabledCount: number; active?: Policy };

export function buildPolicyPosture(policies: Policy[]): PolicyPosture {
  const enabled = policies.filter((policy) => policy.enabled);
  const active = [...enabled].sort((left, right) => right.version - left.version || Date.parse(right.created_at) - Date.parse(left.created_at) || left.id.localeCompare(right.id))[0];
  return { enabledCount: enabled.length, active };
}
```

- [ ] **Step 4: Verify GREEN and commit**

Run: `npm.cmd test -- components/operations/policy-posture-model.test.ts`

Expected: 3 passing tests.

```bash
git add web/components/operations/policy-posture-model.ts web/components/operations/policy-posture-model.test.ts
git commit -m "feat: derive control tower policy posture"
```

### Task 2: Add a semantic policy posture panel

**Files:**
- Create: `web/components/operations/policy-posture.tsx`
- Create: `web/components/operations/policy-posture.test.tsx`

- [ ] **Step 1: Write failing state tests**

```tsx
it("renders configuration facts from the active policy", () => {
  render(<PolicyPosture state={{ kind: "ready", posture: { enabledCount: 1, active: policy } }} />);
  expect(screen.getByRole("heading", { name: "Policy posture" })).toBeInTheDocument();
  expect(screen.getByText("1 enabled in loaded results")).toBeInTheDocument();
  expect(screen.getByText("Production guardrails · v4")).toBeInTheDocument();
  expect(screen.getByText("Rules enabled")).toBeInTheDocument();
  expect(screen.getByText("LLM analysis disabled")).toBeInTheDocument();
});

it("renders a bounded empty state and local retry", async () => {
  const retry = vi.fn();
  const { rerender } = render(<PolicyPosture state={{ kind: "ready", posture: { enabledCount: 0 } }} />);
  expect(screen.getByText("No policies configured")).toBeInTheDocument();
  rerender(<PolicyPosture state={{ kind: "error", onRetry: retry }} />);
  await userEvent.click(screen.getByRole("button", { name: "Retry policy posture" }));
  expect(retry).toHaveBeenCalledOnce();
});
```

- [ ] **Step 2: Verify RED**

Run: `npm.cmd test -- components/operations/policy-posture.test.tsx`

Expected: the panel module does not yet exist.

- [ ] **Step 3: Implement narrow panel props**

```ts
import type { PolicyPosture as Posture } from "./policy-posture-model";

export type PolicyPostureState =
  | { kind: "loading" }
  | { kind: "error"; onRetry: () => void }
  | { kind: "ready"; posture: Posture };
```

Render a labelled section with the exact heading `Policy posture`, a `/settings/policies` link, and either loading/error copy, `No policies configured`, or a definition list with enabled count, active name/version, Rules state, and LLM analysis state. Never use healthy, safe, or compliant.

- [ ] **Step 4: Verify GREEN and commit**

Run: `npm.cmd test -- components/operations/policy-posture.test.tsx`

Expected: 2 passing tests.

```bash
git add web/components/operations/policy-posture.tsx web/components/operations/policy-posture.test.tsx
git commit -m "feat: add policy posture panel"
```

### Task 3: Integrate optional policy data in the control tower

**Files:**
- Modify: `web/components/operations/control-tower.tsx`
- Modify: `web/components/operations/control-tower.test.tsx`
- Modify: `web/app/globals.css`

- [ ] **Step 1: Add failing integration cases**

Mock `listPolicies` from `../../lib/api/policies`. Verify that a user with `policy:read` sees `Production guardrails · v4`; a user without it never calls `listPolicies` and has no Policy posture heading; and one rejected policy request renders only `Policy posture is temporarily unavailable.`, hides the raw error, retries `listPolicies`, and calls `getOverviewData` exactly once.

- [ ] **Step 2: Verify RED**

Run: `npm.cmd test -- components/operations/control-tower.test.tsx`

Expected: no optional request or policy panel exists yet.

- [ ] **Step 3: Add independent state and retry**

```ts
type PolicyState = { kind: "idle" } | { kind: "loading" } | { kind: "error" } | { kind: "ready"; policies: Policy[] };
```

After `getCurrentUser()` passes `agent:read` and `risk:read`, call `listPolicies()` only for `policy:read`. Use the existing cancellation guard. A `policyRequestVersion` effect retries only policies and never increments `requestVersion` or calls `getOverviewData`. Render `PolicyPosture` only for loading/error/ready states after `.tower-evidence` in `.tower-bottom`.

- [ ] **Step 4: Add scoped styles**

```css
.tower-policy-state { min-height: 244px; padding: 20px; border: 1px solid var(--line); border-radius: 14px; background: linear-gradient(145deg, rgba(12, 34, 41, .72), rgba(5, 19, 25, .8)); }
.tower-policy-state dl { display: grid; grid-template-columns: max-content 1fr; gap: 9px 14px; margin: 22px 0 0; font-size: 12px; }
.tower-policy-state dt { color: var(--muted); }
.tower-policy-state dd { margin: 0; color: var(--muted-strong); overflow-wrap: anywhere; }
.tower-policy-error { display: grid; gap: 12px; padding-top: 22px; }
```

Use three lower regions at desktop width and retain the existing single-column stack at `max-width: 900px`.

- [ ] **Step 5: Verify and commit**

Run: `npm.cmd test -- components/operations/policy-posture-model.test.ts components/operations/policy-posture.test.tsx components/operations/control-tower.test.tsx`

Expected: all focused tests pass.

Run: `npm.cmd test -- --reporter=dot && npx.cmd --no-install tsc --noEmit && npm.cmd run build`

Expected: full test suite, types, and production build pass.

Run: `git diff --check`

Expected: no output.

Request focused review of permission gates, error isolation, retry scope, and configuration-only language before merge.

```bash
git add web/components/operations/control-tower.tsx web/components/operations/control-tower.test.tsx web/app/globals.css
git commit -m "feat: show policy posture in control tower"
```

## Self-review

- Every approved requirement has a task: real data, `policy:read`, no-fetch behavior, empty data, isolated error/retry, no compliance claim, and responsive layout.
- No backend/API mutation, compliance score, or policy enforcement change is in scope.
- `PolicyPosture`, `PolicyPostureState`, and `buildPolicyPosture` each have one responsibility.
