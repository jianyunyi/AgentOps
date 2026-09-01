# Phase 3 governance operations

## Member invitations

`POST /api/v1/members/invitations` creates a 48-hour invitation by default. The API returns the raw token only once; only its SHA-256 digest is stored. `POST /api/v1/auth/invitations/accept` atomically creates the user, marks the invitation accepted, creates a session, and appends an audit record. Expired or already-used tokens are rejected.

## Owner safety

`POST /api/v1/members/:id/transfer-owner` atomically demotes the current owner to admin and promotes an active tenant member. Owner roles cannot be edited or disabled, so a tenant cannot be left without an owner through ordinary member operations.

## OIDC / SSO

Set `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL`, and `OIDC_TENANT_ID`. The service discovers the provider, uses Authorization Code + PKCE, signs the state with `SESSION_SECRET`, and verifies the ID Token with the provider's JWKS. Accounts are bound by issuer and subject. The callback is `/api/v1/auth/oidc/callback`.

## Integration verification

Start MySQL and Redis with `docker compose up -d`, then run:

```powershell
$env:AGENTSCOPE_INTEGRATION = "1"
go test ./internal/integration -v
```

The integration suite covers trace/outbox/Redis delivery and the authenticated member API, invitation acceptance, owner transfer, tenant scoping, and audit records.
