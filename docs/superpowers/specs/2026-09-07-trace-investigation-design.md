# Trace Investigation and Risk Disposition Design

## Goal

Turn an active-risk signal into an evidence-backed operator workflow inside the existing trace detail page. An authorized operator must be able to inspect a trace, see only server-redacted risk evidence associated with that trace, and acknowledge or resolve an individual risk event without leaving the page.

This slice closes the product loop `Control Tower → risk event → trace evidence → disposition`. It does not create an incident-management system.

## Scope

- Extend `web/app/dashboard/traces/[traceId]/page.tsx`.
- Load trace detail through the existing `getTrace(traceId)` client.
- Load risk events through the existing `listRiskEvents` client and associate only events whose `trace_id` exactly equals the loaded trace ID.
- Render a trace timeline plus a risk-evidence panel.
- Reuse `reviewRiskEvent(id, status)` for `acknowledged` and `resolved` dispositions.
- Use existing permission data and `StatusBadge` styling.
- Add focused unit/component tests and preserve the current frontend test, TypeScript, and production-build gates.

## Non-goals

- No backend aggregation endpoint, schema migration, or change to existing pagination semantics.
- No owner assignment, comments, SLA, escalation policy, bulk review, or incident record.
- No direct display of raw prompt text, unredacted event payloads, or span input/output snapshots.
- No inferred agent heartbeat, system safety, compliance, or historical completeness claim.

## Architecture

The trace detail page remains the composition boundary. It obtains the current user through the existing session client, then applies permissions independently:

1. Trace detail remains the primary request and continues to own the page-level loading and failure state.
2. A user with `risk:read` also loads the existing paginated risk-event result set.
3. A pure model function associates, orders, and exposes only the events for the loaded trace.
4. `TraceRiskEvidence` renders the derived state and may issue a review mutation only when the user has `risk:review`.

The page must not request risk events when the session lacks `risk:read`, and must not render review controls without `risk:review`. The trace timeline is still available to a user allowed to read the trace even if the risk request fails or is forbidden.

## Data and presentation

### Association and ordering

The model receives `RiskEvent[]` and `traceId` and returns events with an exact `event.trace_id === traceId` match. It orders results newest-first by valid `created_at`. Invalid or missing timestamps sort after valid timestamps; ties use lexical event ID ordering for deterministic output.

The UI shows risk level, rule code, detector, reason, redacted evidence, event status, and creation time. It does not use `Span.inputSnapshot` or `Span.outputSnapshot` as evidence.

### Layout

On desktop, the page is a trace-investigation grid: execution timeline is the primary region and evidence is a constrained secondary panel. On narrow screens it becomes a single column, with evidence following the timeline. Existing console colors and status badges communicate state; no additional decorative 3D or animated visual is in this workflow.

When the associated result set is empty, the panel says: `No associated risk events in loaded results.` This is intentionally bounded: it is not a statement that the trace or system is safe.

### Review actions

`acknowledge` and `resolve` call the existing review endpoint with `acknowledged` or `resolved` respectively. While an event mutation is in flight, actions for that event are disabled. On success, update only that event in local state. Do not refetch the trace or risk list. On failure, preserve the prior status and show a panel-local error.

## Loading, error, and race behavior

- Trace request failure retains the current page-level error state because there is no meaningful trace context to render.
- Risk request failure must be isolated to the risk-evidence panel. The trace timeline remains visible.
- The risk panel exposes a retry that repeats only the risk request; it does not reload the trace.
- Request cancellation/ignore guards prevent an older route or retry response from overwriting state for the current trace.
- Switching trace routes clears stale risk-event state before the next result is applied.

## Accessibility

- The evidence region has a named heading and loading/error/empty states use appropriate semantic status or alert roles.
- Action buttons have explicit labels including the disposition and the event context when needed.
- The design remains keyboard-operable; disabled state is semantic, not color-only.

## Test plan

Add tests for:

1. exact trace association, deterministic newest-first ordering, invalid-date handling, and no-result behavior;
2. rendering redacted evidence but not arbitrary span snapshots;
3. no risk request/panel without `risk:read`;
4. read-only evidence with `risk:read` but without `risk:review`;
5. successful acknowledgement and resolution update only the target event;
6. failed review preserves status and reports a local error;
7. risk-load failure preserves the trace timeline and retry repeats only the risk request;
8. mobile-safe layout styling and normal frontend quality gates.

## Acceptance criteria

- An authorized operator can inspect a trace and its related, redacted risk evidence on one page.
- An operator with `risk:review` can acknowledge or resolve individual events, with visible pending and error states.
- A trace without events, a forbidden risk permission, and a risk-fetch failure all make bounded, non-deceptive claims while preserving usable trace investigation.
- The change adds no endpoint or data model beyond existing clients and does not weaken session authorization.
