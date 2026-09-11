# Windows Self-Hosted Installer Design

## Goal

Deliver AgentOps as an open-source Windows self-hosted website: an operator installs one signed Windows package, completes a short configuration flow, and uses a local control tool to start, stop, diagnose, update, back up, and open the browser console. The operator must not install Go, Node.js, npm, MySQL, or Redis manually.

## Scope and constraints

- Target: Windows 10/11 x64 first.
- Required external platform: Docker Desktop with WSL 2 enabled. The installer validates it but does not redistribute Docker Desktop or silently enable virtualization.
- Runtime: existing API, Worker, MySQL, Redis, and Next.js web console run as Docker Compose services.
- Optional local LLM: Ollama is enabled only by explicit configuration and runs through the existing local-LLM Compose profile.
- No production cloud deployment, Kubernetes, or Azure provisioning belongs to this package.
- Secrets stay in an ignored configuration file on the local machine and must never appear in the installer source, logs, or generated Compose files.

## User experience

1. The user downloads `AgentOps-Setup-x64.exe` from a GitHub Release and launches it.
2. The installer checks Windows architecture, free disk space, Docker Desktop, WSL 2, and Docker Compose availability. If Docker Desktop is stopped, the user receives a precise remediation message.
3. The installer writes the application files under `%ProgramFiles%\AgentOps` and creates data/config directories under `%ProgramData%\AgentOps`.
4. A configuration wizard asks for public web/API ports, administrator bootstrap identity, MySQL password, and whether to enable the Ollama profile. It generates session and agent-signing secrets locally with cryptographic randomness.
5. The installer runs `agentopsctl configure` and `agentopsctl start`. Docker Compose creates named volumes for MySQL and Redis, then starts API, Worker, and Web.
6. A post-install health check waits for API readiness and opens `http://127.0.0.1:<web-port>` only after the stack is healthy.

The Start Menu exposes AgentOps Control, Start, Stop, Logs, Backup, Diagnose, and Uninstall. The control program invokes only the bundled Compose files and never evaluates configuration as shell code.

## Runtime architecture

```
Windows installer
  ├─ %ProgramFiles%\AgentOps
  │    ├─ compose.yaml + images/version metadata
  │    └─ agentopsctl.exe
  └─ %ProgramData%\AgentOps
       ├─ config\agentops.env       (operator-only secrets and settings)
       ├─ data\                     (Docker volumes / backup exports)
       └─ logs\install.log          (redacted diagnostics)

agentopsctl.exe
  ├─ validates Docker Desktop and Compose
  ├─ reads a strict KEY=value env file
  ├─ invokes `docker compose --env-file ... -f ...`
  └─ probes web/API health endpoints

Docker Compose
  ├─ web: Next.js standalone server → localhost web port
  ├─ api: Gin API → localhost API port
  ├─ worker: asynchronous outbox/risk processing
  ├─ mysql: persistent named volume
  └─ redis: persistent named volume
```

The Compose project name is fixed to `agentops`. Ports bind to `127.0.0.1` by default. Exposing a LAN address is an explicit advanced setting and is not enabled by default.

## Configuration contract

The wizard and CLI produce `%ProgramData%\AgentOps\config\agentops.env` with restrictive Windows ACLs. It contains only declared scalar values; the parser rejects newlines, duplicate keys, unknown keys, and shell-expansion syntax.

Required settings:

- `WEB_PORT` and `API_PORT`, validated as available local TCP ports.
- `MYSQL_PASSWORD`, stored only in the local env file.
- generated `SESSION_SECRET` and `AGENT_SIGNING_ENCRYPTION_KEY`.
- `WEB_ORIGIN`, derived from `WEB_PORT`.

Optional settings:

- `ENABLE_LOCAL_LLM=false|true` and `LLM_MODEL` when local Ollama is selected.
- bounded Worker retry and metrics values already supported by the application.

The installer must not collect telemetry or transmit configuration. It must display the exact configuration file location and backup location.

## Lifecycle and recovery

- `start`: validates prerequisites, validates config, performs `docker compose up -d`, waits for MySQL/Redis/API/Web health, and prints local URLs.
- `stop`: performs `docker compose stop`; volumes remain intact.
- `logs`: streams selected redacted container logs without printing the env file.
- `diagnose`: reports Docker/Compose versions, container status, health endpoint response, disk availability, and redacted configuration validity.
- `backup`: exports MySQL through a short-lived Compose command and records the image version; Redis persistence remains in its named volume. Restore is a separate explicit confirmation command.
- `update`: pulls a release-specific immutable image manifest, validates its version, reapplies the existing env file, and runs health checks. It never overwrites secrets or volumes.
- `uninstall`: offers to preserve data by default. Removing volumes requires a separate destructive confirmation.

## Packaging and release

- Build API/Worker images from the existing Dockerfile and Web as a standalone Next.js image.
- Publish immutable versioned images to GHCR, plus a signed release manifest containing image digests.
- Build `agentopsctl.exe` as a small Go program for Windows amd64.
- Package the runtime files and CLI with Inno Setup into `AgentOps-Setup-x64.exe`.
- GitHub Actions produces the installer, SBOM/checksums, and GitHub Release artifacts only after Go tests, frontend tests/build, Compose validation, and Windows package smoke tests pass.

## Security model

- Docker Desktop and WSL 2 remain user-managed prerequisites.
- Local ports default to loopback; no automatic firewall rules or public exposure.
- Docker named volumes own database persistence; secrets are not embedded in images.
- Installer and CLI redact `*_PASSWORD`, `*_SECRET`, `*_KEY`, and DSN values from diagnostics.
- Update metadata is digest-pinned; floating image tags are not accepted by the control tool.
- The local first-admin bootstrap flow uses the existing registration/tenant initialization API and must be one-time/idempotent.

## Acceptance criteria

1. A clean supported Windows VM with Docker Desktop/WSL 2 can install the released EXE without Go, Node, npm, MySQL, or Redis.
2. The wizard creates a valid protected configuration file without exposing generated secrets.
3. `agentopsctl start` brings up Web, API, Worker, MySQL, and Redis; health checks pass and the browser console opens.
4. Stop/start preserves data; backup creates a recoverable database export; uninstall preserves data unless the user explicitly opts in to deletion.
5. Invalid Docker state, port conflict, invalid config, failed container health, and failed upgrade produce actionable errors without leaking secrets.
6. Release artifacts contain checksums/SBOM and use immutable image digests.

## Explicit non-goals for the first release

- Fully native bundled MySQL/Redis Windows services.
- macOS and Linux installers.
- Automatic TLS certificates, domain configuration, or public internet exposure.
- Automatic Docker Desktop, WSL, or Windows feature installation.
- Managed cloud deployment.
