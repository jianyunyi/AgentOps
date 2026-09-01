# P0 Authorization Boundary Hardening

## Goal

Close the two production-blocking authorization gaps identified in the 2026-09-01 audit: stale browser sessions retaining old roles and Agent API keys reading unrelated tenant traces.

## Design

Session resolution remains database-backed. After loading a non-expired session, the service loads the current user by `session.UserID`, rejects disabled or cross-tenant users, and replaces the session role with the current database role. Logout revokes the server-side session before clearing the cookie. Reusing an old cookie after a role/status change therefore fails or receives the current permissions.

Trace query authentication is explicit. A console session must pass the existing `agent:read` permission before querying. An Agent API key may query only traces whose `agent_id` equals the authenticated Agent ID. Both paths retain tenant predicates in every repository query. Ingestion continues to use Agent Key authentication and is unchanged.

## Error handling

Missing, expired, revoked, disabled, or cross-tenant sessions return `401`. A valid session without `agent:read` returns `403`. An Agent Key cannot widen its query scope because the Agent ID filter is supplied by server-side authentication context, never by a request parameter.

## Verification

Add regression tests for current-role/session invalidation, server-side logout revocation, console permission enforcement, and Agent-Key self-only Trace queries. Run the complete Go suite, `go vet`, frontend tests/build, and CI.
