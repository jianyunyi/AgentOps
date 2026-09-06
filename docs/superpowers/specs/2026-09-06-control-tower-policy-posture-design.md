# Control Tower Policy Posture Design

## Purpose

Show the current tenant's real policy configuration in the AgentOps control tower without turning policy state into a claim about runtime compliance.

## Data and permissions

The existing `GET /api/v1/policies` endpoint returns the tenant-scoped `Policy[]` contract. The control tower will request it only after the authenticated user has passed the existing `agent:read` and `risk:read` gate and also holds `policy:read`.

The existing Agent, Trace, and risk overview remains the primary request. Policy state is an optional, independent request:

- No `policy:read`: no policy request and no policy panel.
- `policy:read` with an empty list: render `No policies configured`.
- A policy request failure: keep the control tower visible and render a bounded `Policy posture is temporarily unavailable.` message with a panel-local Retry action.
- A successful list: derive enabled count from `policy.enabled`, choose the active policy as the enabled policy with the largest `version` (then newest `created_at`, then lexical `id` as deterministic tie-break), and show its `rules_enabled` and `llm_enabled` flags.

The UI never calls a policy enabled/disabled state "healthy", "safe", or "compliant". It presents configuration only. The list is authoritative only for the loaded tenant response; it does not infer enforcement results.

## UI

Add a `PolicyPosture` panel in the existing lower control-tower region. It contains:

- eyebrow: `Governance configuration`
- heading: `Policy posture`
- a link to `/settings/policies`
- enabled count in loaded results
- active policy name and version when one exists
- textual Rules and LLM analysis state for the active policy

The panel uses semantic headings, definition list content, a live-status region only for loading/retry state, and existing control-tower visual tokens. It does not add a new global dashboard KPI or fake a policy row when no data is available.

## Component boundaries

- `web/components/operations/policy-posture-model.ts`: pure `Policy[]` to view model conversion; no React or network logic.
- `web/components/operations/policy-posture.tsx`: presentational loading, empty, error, and ready panel with a supplied retry callback.
- `web/components/operations/control-tower.tsx`: owns the authenticated policy fetch and passes state into `PolicyPosture`.

Keeping policy fetch state separate prevents an optional `GET /policies` failure from taking down the Agent operations view and prevents a retry from refetching unrelated live records.

## Testing

Unit tests cover enabled count, deterministic active-policy selection, and the empty list. Component tests cover successful data, no permission/no fetch, empty data, bounded failure and retry. Existing control-tower tests continue to cover the primary loading, error, and permission path.

## Scope limits

No backend endpoint, database schema, policy mutation, or policy enforcement logic changes. No dashboard-wide policy history or compliance score is introduced.
