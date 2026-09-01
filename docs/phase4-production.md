# Phase 4 production operations

## Runtime baseline

- API server uses read-header, read, write, and idle timeouts.
- SIGINT/SIGTERM triggers a 15-second graceful shutdown.
- MySQL pool limits are configurable with `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and `DB_CONN_MAX_LIFETIME_MINUTES`.
- `SESSION_SECRET` must contain at least 32 characters.

## CI gates

The backend CI job starts disposable MySQL 8.4 and Redis 7.4 services, runs the unit suite and `go vet`, then executes the real integration suite with `AGENTSCOPE_INTEGRATION=1`.

## Backup and restore

Use `scripts/backup-restore-check.ps1` to create a consistent logical backup. Restore into a disposable database and verify migration metadata, tenant/user rows, audit rows, outbox state, and risk events before declaring a recovery drill successful. Never test restore by overwriting the production database.

## Release gate

Before production deployment, verify: migration succeeds on a copy, health readiness is green, Redis stream consumption is active, outbox pending/dead counts are monitored, error rate and latency alerts are configured, backup restore has a timestamped record, and rollback image/configuration is available.
