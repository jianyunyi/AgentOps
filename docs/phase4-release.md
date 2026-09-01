# Phase 4 release, capacity, and recovery

## Canary and rollback

Build immutable image tags from the Git commit. Deploy `deploy/k8s/api-canary.yaml` with one replica, route a small percentage of traffic through the canary service using the ingress controller, and compare error rate/latency for at least 10 minutes. Promote by updating the stable Deployment image. Roll back immediately with `kubectl rollout undo deployment/agentscope-api -n agentscope` or by restoring the previous immutable image tag.

## SLOs

- API availability: 99.9% monthly, excluding planned maintenance.
- API average latency: less than 1 second for the baseline request path.
- Outbox delivery: 99.99% of events delivered within 60 seconds.
- Risk analysis: 99% completed within 2 minutes; failed LLM analysis must fall back to deterministic rules.

Use the Prometheus rules in `deploy/prometheus/alerts.yml` as the initial alert baseline. Production should add recording rules, notification routing, and dashboards for the SLO error budget.

## Capacity test

Run `scripts/load-test.ps1` against `/health/live` first, then against an authenticated read endpoint in a disposable environment. Record RPS, p50/p95/p99 latency, error rate, CPU, memory, MySQL connections, Redis latency, and outbox lag. Increase concurrency until the first SLO is violated; that point is the initial capacity limit, not a production target.

## Security gates

CI runs `govulncheck` for Go dependencies and `npm audit --audit-level=high` for frontend dependencies. High-severity findings block release unless an approved exception is recorded with an owner and expiry date.
