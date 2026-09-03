# AgentOps Console UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the AgentOps console shell and root Overview with a memorable Agent network visualization, fluid motion, and production-safe states while preserving existing routes, API contracts, authentication, and permissions.

**Architecture:** Keep static page composition in Server Components and isolate browser-only behavior in small Client Components. The root page uses a shared console shell, real API-backed signal cards, and a dependency-free CSS 3D network scene. Existing dense screens reuse the same tokens and state primitives without adopting spatial navigation.

**Tech Stack:** Next.js 16, React 19, TypeScript, native CSS design tokens with CSS 3D transforms and animations, Vitest, Testing Library.

---

## File Map

- Create `web/app/page.tsx`: authenticated console Overview composition.
- Create `web/components/layout/console-shell.tsx`: shared navigation, tenant context, and sign-out action.
- Create `web/components/layout/console-shell.test.tsx`: navigation and sign-out behavior.
- Create `web/components/landing/agent-network-scene.tsx`: isolated CSS 3D scene with keyboard interaction and semantic fallback.
- Create `web/components/landing/agent-network-scene.test.tsx`: fallback, labels, reduced-motion, and navigation tests.
- Create `web/components/landing/overview-signals.tsx`: client data loader for agents, traces, risk events, and permission-aware states.
- Create `web/components/landing/overview-signals.test.tsx`: loading, success, empty, failure, and permission states.
- Create `web/components/ui/status-badge.tsx`: consistent status rendering.
- Create `web/components/ui/state-view.tsx`: loading, empty, error, and forbidden states.
- Modify `web/app/layout.tsx`: use the shell and update document metadata/language.
- Modify `web/app/globals.css`: replace starter CSS with the approved token system, responsive layout, fluid background, table primitives, and reduced-motion rules.
- Modify `web/app/login/page.tsx`: apply the new auth composition and explicit error/loading state.
- Modify `web/app/dashboard/traces/page.tsx`, `web/app/dashboard/risk/page.tsx`, `web/app/dashboard/traces/[traceId]/page.tsx`, `web/app/settings/agents/page.tsx`, `web/app/settings/members/page.tsx`, `web/app/settings/policies/page.tsx`, and `web/app/settings/audit/page.tsx`: wrap content in the shared shell and replace ad hoc visual classes with the common primitives without changing API calls or permission checks.

### Task 1: Add the shared visual foundation

**Files:**
- Create `web/components/ui/status-badge.tsx`
- Create `web/components/ui/state-view.tsx`
- Create `web/components/layout/console-shell.tsx`
- Create `web/components/layout/console-shell.test.tsx`
- Modify `web/app/layout.tsx`
- Modify `web/app/globals.css`

- [ ] **Step 1: Write failing tests for shell and state primitives**

```tsx
it("renders the primary routes and tenant context", () => {
  render(<ConsoleShell><div>content</div></ConsoleShell>);
  expect(screen.getByRole("link", { name: "Overview" })).toHaveAttribute("href", "/");
  expect(screen.getByRole("link", { name: "Traces" })).toHaveAttribute("href", "/dashboard/traces");
  expect(screen.getByText("Tenant workspace")).toBeInTheDocument();
});

it("renders accessible error and forbidden states", () => {
  render(<StateView kind="error" title="Could not load" description="Retry the request." />);
  expect(screen.getByRole("alert")).toHaveTextContent("Could not load");
});
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run: `npm test -- --run components/layout/console-shell.test.tsx` from `web`.

Expected: FAIL because the shell and state primitives do not exist.

- [ ] **Step 3: Implement the primitives and shell**

Use a typed `StateView` union of `loading | empty | error | forbidden`, render `role="status"` for loading/empty, `role="alert"` for error/forbidden, and accept an optional action label/callback. `ConsoleShell` must render `children`, a single desktop-line nav, the current tenant label, and a `Sign out` button that calls `POST /api/v1/auth/logout` through the existing request helper before redirecting to `/login`. Do not read or display credentials.

- [ ] **Step 4: Replace the starter CSS with the approved token system**

Define CSS variables for the deep blue-black background, cyan primary, orange risk, indigo trace, text, muted text, border, radius, spacing, focus ring, and shadows. Add `.app-shell`, `.app-header`, `.app-nav`, `.page-frame`, `.hero-grid`, `.signal-grid`, `.panel`, `.button-primary`, `.button-quiet`, `.status-badge`, table, form, and state classes. Add `@media (prefers-reduced-motion: reduce)` to remove transforms, transitions, and animation names. Avoid `h-screen`, em-dash copy, default purple gradients, and blanket card repetition.

- [ ] **Step 5: Run the focused tests and commit**

Run: `npm test -- --run components/layout/console-shell.test.tsx`.

Expected: PASS.

Commit: `git add web/app/layout.tsx web/app/globals.css web/components/layout web/components/ui && git commit -m "feat: add AgentOps console visual foundation"`

### Task 2: Build the Overview page with real signals

**Files:**
- Create `web/app/page.tsx`
- Create `web/components/landing/overview-signals.tsx`
- Create `web/components/landing/overview-signals.test.tsx`
- Create `web/lib/api/overview.ts`

- [ ] **Step 1: Write failing tests for signal states**

```tsx
it("shows the overview signal summary from API data", async () => {
  vi.mock("../../lib/api/overview", () => ({ getOverviewData: vi.fn().mockResolvedValue({ agents: [agent], traces: [trace], risks: [risk] }) }));
  render(<OverviewSignals />);
  expect(await screen.findByText("1 active Agent")).toBeInTheDocument();
  expect(screen.getByText("1 open risk")).toBeInTheDocument();
});

it("does not invent metrics when the API is empty", async () => {
  vi.mocked(getOverviewData).mockResolvedValue({ agents: [], traces: [], risks: [] });
  render(<OverviewSignals />);
  expect(await screen.findByText("No live signals yet")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `npm test -- --run components/landing/overview-signals.test.tsx` from `web`.

Expected: FAIL because the overview loader and component do not exist.

- [ ] **Step 3: Implement the real-data adapter**

`getOverviewData` must call `listAgents()`, `listTraces(1, 5)`, and `listRiskEvents(1, "open")` using existing request functions. Return `{ agents, traces, risks }`. The component must call `getCurrentUser()` first, use `hasPermission` for `agent:read` and `risk:read`, and render explicit forbidden state for unavailable data. It may calculate counts from returned arrays only; it must not fabricate percentages, cost, or realtime claims.

- [ ] **Step 4: Implement the server-rendered root composition**

`web/app/page.tsx` must render `ConsoleShell` with a two-column hero: left title/copy/CTAs and right `AgentNetworkScene`. Below it render four real route links for Trace Explorer, Risk Review, Policy Guardrails, and Audit Evidence, then `OverviewSignals`. Use Chinese product copy approved in the design and keep visible copy free of em-dash characters.

- [ ] **Step 5: Run focused tests and commit**

Run: `npm test -- --run components/landing/overview-signals.test.tsx`.

Expected: PASS.

Commit: `git add web/app/page.tsx web/components/landing web/lib/api/overview.ts && git commit -m "feat: add AgentOps control room overview"`

### Task 3: Add the Agent network 3D scene and fallback

**Files:**
- Create `web/components/landing/agent-network-scene.tsx`
- Create `web/components/landing/agent-network-scene.test.tsx`

- [ ] **Step 1: Write failing tests for the 3D scene and accessible nodes**

```tsx
it("renders the static network fallback when WebGL is unavailable", () => {
  vi.mock("three", () => ({ WebGLRenderer: vi.fn(() => { throw new Error("no webgl"); }) }));
  render(<AgentNetworkScene />);
  expect(screen.getByRole("img", { name: /Agent network/i })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /Trace/i })).toBeInTheDocument();
});

it("navigates a focused semantic node on Enter", async () => {
  render(<AgentNetworkScene />);
  await userEvent.click(screen.getByRole("button", { name: /Trace/i }));
  expect(mockedRouter.push).toHaveBeenCalledWith("/dashboard/traces");
});
```

- [ ] **Step 2: Run the tests and verify they fail**

Run: `npm test -- --run components/landing/agent-network-scene.test.tsx`.

Expected: FAIL because the scene component does not exist.

- [ ] **Step 3: Implement the isolated client scene**

Add `"use client"`. Define a typed topology array with `agent`, `trace`, `policy`, `risk`, and `audit` nodes. Render the topology as a perspective CSS 3D stage with orbit rings, depth, a faceted core, edges, and a small number of animated flow pulses. Expose semantic HTML buttons for keyboard and screen-reader access and use `useRouter` for explicit navigation. Keep the scene dependency-free.

- [ ] **Step 4: Add reduced-motion and responsive behavior**

Read `window.matchMedia("(prefers-reduced-motion: reduce)")` inside the client component. In reduced-motion mode, render the scene without camera drift or particle animation. At mobile widths, use a compact scene height and keep buttons in a scrollable accessible list below the canvas. Ensure the component does not use `window.addEventListener("scroll")`.

- [ ] **Step 5: Run focused tests and commit**

Run: `npm test -- --run components/landing/agent-network-scene.test.tsx`.

Expected: PASS.

Commit: `git add web/components/landing/agent-network-scene.tsx web/components/landing/agent-network-scene.test.tsx && git commit -m "feat: add Agent network visualization"`

### Task 4: Upgrade login and existing console routes

**Files:**
- Modify `web/app/login/page.tsx`
- Modify `web/app/dashboard/traces/page.tsx`
- Modify `web/app/dashboard/risk/page.tsx`
- Modify `web/app/dashboard/traces/[traceId]/page.tsx`
- Modify `web/app/settings/agents/page.tsx`
- Modify `web/app/settings/members/page.tsx`
- Modify `web/app/settings/policies/page.tsx`
- Modify `web/app/settings/audit/page.tsx`

- [ ] **Step 1: Preserve behavior with route smoke tests**

Run existing tests before editing: `npm test -- --run` from `web`.

Expected: all existing tests pass. Record the baseline count in the task notes.

- [ ] **Step 2: Wrap every page in the shared shell**

Keep each existing API call, form field name/order, permission string, route path, and mutation behavior unchanged. Add `ConsoleShell` around page content, replace raw `panel`, `error-state`, and `button` usage with the common classes/components, and add clear empty/loading/error/forbidden states where a page currently renders an empty table without explanation.

- [ ] **Step 3: Improve login state handling**

Keep email and password inputs in the same order and preserve the SSO path. Add a disabled submit state while `login` is pending, `aria-live="polite"` status text, a retry-safe error message, and a visible focus ring. Do not echo credentials or server response bodies.

- [ ] **Step 4: Add route-specific status semantics**

Use `StatusBadge` for trace status, risk level, agent status, member status, and policy status. Add explicit permission-denied views for Agents, Members, and Policies based on the existing `CurrentUser.permissions` values. Keep audit and risk evidence readable in narrow screens by allowing only the table container to scroll.

- [ ] **Step 5: Run tests and commit**

Run: `npm test -- --run` from `web`.

Expected: all baseline tests plus new UI tests pass.

Commit: `git add web/app web/components web/lib && git commit -m "feat: unify AgentOps console routes"`

### Task 5: Production verification and visual QA

**Files:**
- Modify only files needed to fix verified failures from Tasks 1 to 4.

- [ ] **Step 1: Run typecheck and unit tests**

Run from `web`: `npx tsc --noEmit` and `npm test -- --run`.

Expected: no TypeScript diagnostics and all tests pass.

- [ ] **Step 2: Run the production build**

Run from `web`: `npm run build`.

Expected: Next.js production build completes successfully with no client/server boundary errors.

- [ ] **Step 3: Run dependency audit**

Run from `web`: `npm audit --audit-level=high`.

Expected: no high or critical vulnerabilities introduced by the UI changes.

- [ ] **Step 4: Perform manual visual QA**

Start the console with `npm run dev` and check `/`, `/login`, `/dashboard/traces`, `/dashboard/risk`, `/settings/agents`, `/settings/members`, `/settings/policies`, and `/settings/audit` at desktop and mobile widths. Verify keyboard focus, Enter navigation in the network, reduced motion, loading, empty, error, and permission-denied states. Confirm no API key, signing secret, or sensitive payload appears in browser logs.

- [ ] **Step 5: Final commit**

Run `git diff --check` and `git status --short`.

Expected: no whitespace errors and only intentional changes remain.

Commit: `git add web && git commit -m "chore: verify AgentOps console UI release"`

## Plan Self-Review

- Spec coverage: visual tokens, Control Room layout, CSS 3D topology, fluid motion, semantic fallback, accessibility, API-backed signals, stable routes, security constraints, and verification are covered by Tasks 1 through 5.
- Completeness scan: no unfinished work markers or vague implementation steps are used in the plan.
- Type consistency: `OverviewSignals` consumes `getOverviewData` with `{ agents, traces, risks }`; `AgentNetworkScene` consumes the typed topology and exposes node navigation; `ConsoleShell` owns navigation and sign-out.
- Scope check: no backend endpoint, database schema, auth contract, or full-app spatial navigation is introduced.
