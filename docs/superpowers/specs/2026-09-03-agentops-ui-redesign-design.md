# AgentOps Console UI Redesign Design

Date: 2026-09-03
Status: Approved for implementation planning

## Design Read

This is a redesign of an enterprise AgentOps control console for technical teams, security engineers, and platform owners. The visual language is a deep-sea dark control room with electric cyan, signal orange, and indigo accents. The product story is that every AI Agent action can be observed, governed, and audited.

## Product Intent

The console needs a memorable first impression without compromising daily operational work. The new root page will explain the product's core value through an Agent network visualization, while Trace, Risk, Agent, Member, Policy, and Audit pages remain evidence-first workspaces with shared navigation and state handling.

## Chosen Approach

Use the `Control Room` approach. The root page is a product-level Overview with a left narrative column and a right Agent network scene. Existing business routes remain stable and are upgraded to share the same shell, visual tokens, controls, and loading/error/empty states. Three-dimensional motion is reserved for product explanation and network topology, not dense data tables.

## Visual System

- Design variance: 8
- Motion intensity: 7
- Visual density: 4
- Single dark theme across the page
- Primary accent: `#67F4D2` electric cyan for healthy Agents, connections, and primary actions
- Risk accent: `#FF966B` signal orange for risk nodes and alerts
- Secondary accent: `#789BFF` indigo for traces and data flow
- Deep blue-black background with layered radial gradients, avoiding AI-purple defaults
- UI typography uses a readable sans-serif system stack with restrained serif display headings
- Consistent medium radius system and visible focus rings
- No em-dash characters in visible UI copy

## Information Architecture

The root `/` page will include:

1. A single-line desktop navigation with brand, Overview, Traces, Risk, Agents, Settings, tenant context, and sign-out.
2. A hero control room with the headline `让每个 Agent 的行为都能被看见`, a concise explanation, a Trace Explorer primary CTA, and an Agents secondary CTA.
3. A right-side Agent network visualization with Agent, Trace, Policy, Risk, and Audit semantics.
4. A business capability section for Trace Explorer, Risk Review, Policy Guardrails, and Audit Evidence, each linked to an existing route.
5. A Recent Signal area for recent traces, risks, and audit status backed by real API data. Loading, empty, and error states are explicit.

Existing routes keep their URL paths and API contracts. They will share the same shell and UI tokens where practical.

## Agent Network Scene

The visualization is a low-poly Three.js scene generated from domain semantics rather than a decorative external model. Nodes represent Agents, Traces, Policies, Risks, and Audit evidence. Edges represent operational relationships and animated light trails represent event flow.

The scene must:

- Render in an isolated client component: `web/components/landing/agent-network-scene.tsx`.
- Accept a small typed topology model so real API data can be connected later without changing rendering code.
- Support pointer hover, keyboard focus, and Enter navigation for semantic nodes.
- Keep hover informational and navigation explicit.
- Use `requestAnimationFrame` with cancellation and dispose all Three.js resources on unmount.
- Use a static 2D fallback when WebGL is unavailable.
- Reduce or stop nonessential movement under `prefers-reduced-motion`.
- Avoid logging secrets or sensitive event payloads in the browser.

## Fluid Motion

Motion will be used to communicate hierarchy, realtime flow, and state transitions:

- One-time hero reveal for the narrative and scene.
- Slow, bounded camera drift and node orbit while motion is allowed.
- Short event-flow pulses across selected edges.
- Subtle hover/focus response on nodes and actions.

No continuous animation will be applied to dense tables or every card. The animation layer must not block interaction or create layout shift.

## Component Boundaries

- `web/app/page.tsx`: server-rendered root page composition and static copy.
- `web/components/layout/console-shell.tsx`: shared navigation, tenant context, and sign-out affordance.
- `web/components/landing/agent-network-scene.tsx`: client-only 3D scene, fallback, and node interaction.
- `web/components/ui/*`: shared buttons, panels, status badges, and state views.
- `web/app/globals.css`: design tokens, layout primitives, backgrounds, responsive rules, and reduced-motion styles.

Interactive or animation-heavy code must stay in client leaf components. Static layout remains server-rendered.

## Responsive and Accessibility Rules

- Desktop uses an asymmetric two-column hero with the 3D scene on the right.
- Below tablet width, content stacks with copy first and the scene second.
- On mobile, the scene uses a compact static 2D network summary if performance or WebGL support is insufficient.
- Every semantic node has an accessible label and keyboard focus state.
- Buttons and form controls maintain readable contrast and never wrap their primary labels.
- `prefers-reduced-motion` removes camera drift, pulses, and entrance transforms while preserving state and hierarchy.
- Loading, empty, error, and permission-denied states are visible and actionable.

## Data and Security Constraints

- Existing auth, permission checks, URL paths, API field names, and form field ordering are not changed.
- The root page may use existing read APIs, but it must not invent production metrics when data is absent.
- Agent API keys and signing secrets never enter client logs, scene metadata, or static markup.
- API failures are surfaced through bounded user-facing errors and do not silently disappear.

## Verification

- Run the existing web unit tests and add coverage for the root page, node navigation, and fallback rendering.
- Run TypeScript type checking and the Next.js production build.
- Manually verify desktop and mobile layouts, keyboard navigation, WebGL fallback, reduced motion, loading, empty, error, and permission-denied states.
- Run the Go regression suite because shared API contracts remain in scope.

## Out of Scope

- No new backend endpoints or schema changes.
- No changes to authentication behavior or permission semantics.
- No full spatial navigation for the entire application.
- No external 3D asset pipeline or model marketplace dependency.
