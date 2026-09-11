# Windows Self-Hosted Installer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Windows x64 installer that runs the complete AgentOps website locally using Docker Desktop and a testable control CLI.

**Architecture:** Local Docker Compose runs Web, API, Worker, MySQL, and Redis. A strict Go CLI manages configuration and lifecycle; Inno Setup installs immutable runtime files in Program Files and local state in ProgramData.

**Tech Stack:** Go 1.25, Docker Compose, Next.js standalone, PowerShell, Inno Setup 6, GitHub Actions.

---

### Task 1: Compose-based local website

**Files:** Create `web/Dockerfile`, `deploy/windows/compose.yaml`, `deploy/windows/agentops.env.example`, `scripts/validate-windows-package.ps1`, `scripts/validate-windows-package.test.ps1`; modify `web/next.config.ts`.

- [ ] Write a failing PowerShell contract test requiring Web/API/Worker/MySQL/Redis services; loopback-only Web/API ports; health checks; no inline secrets; and no floating image tags.
- [ ] Run `powershell.exe -ExecutionPolicy Bypass -File scripts\validate-windows-package.test.ps1`; expect failure because files are absent.
- [ ] Set Next `output: 'standalone'`. Add a multi-stage Web Dockerfile (`npm ci`, `npm run build`, standalone/static runtime copy). Add Compose with digest image variables, named data volumes, loopback mappings, and a secret-free env example.
- [ ] Implement the validator and rerun the test; expect `Windows package validation tests passed.`
- [ ] Commit: `git add web/Dockerfile web/next.config.ts deploy/windows scripts/validate-windows-package*.ps1` then `git commit -m "feat: add Windows local Compose stack"`.

### Task 2: Strict config and control CLI

**Files:** Create `internal/selfhost/config.go`, `internal/selfhost/config_test.go`, `internal/selfhost/compose.go`, `internal/selfhost/compose_test.go`, `cmd/agentopsctl/main.go`.

- [ ] Write failing `ParseConfig` tests: accept declared scalar keys; derive `WEB_ORIGIN`; use `crypto/rand` to generate 32-byte base64 session/signing secrets; reject duplicate, unknown, multiline, shell-expansion, non-loopback, and invalid-port values.
- [ ] Run `go test ./internal/selfhost -run TestParseConfig -count=1`; expect package-not-found failure.
- [ ] Implement a whitelist for Web/API ports, MySQL password, session/signing secrets, local-LLM settings, and existing Worker settings. Redact password/secret/key/DSN fields in diagnostics.
- [ ] Write failing `BuildComposeArgs` tests: always provide project name `agentops`, one validated env file, `deploy/windows/compose.yaml`, and permit only fixed lifecycle/backup operations.
- [ ] Run `go test ./internal/selfhost -run TestBuildComposeArgs -count=1`; expect missing-function failure.
- [ ] Implement Docker Desktop/Compose checks, readiness polling, and commands `configure`, `start`, `stop`, `status`, `logs`, `diagnose`, `backup`. Backup writes timestamped SQL below ProgramData and never deletes volumes.
- [ ] Run `go test ./internal/selfhost ./cmd/agentopsctl`; expect PASS. Commit as `feat: add Windows self-hosted control CLI`.

### Task 3: Inno Setup package

**Files:** Create `installer/AgentOps.iss`, `installer/scripts/agentops-start.cmd`, `installer/scripts/agentops-stop.cmd`, `installer/scripts/agentops-logs.cmd`, `installer/scripts/agentops-diagnose.cmd`; modify package validator scripts.

- [ ] Extend the failing contract test: x64 only, Docker Desktop/Compose validation, `{autopf}\AgentOps`, `{commonappdata}\AgentOps`, no credentials, fixed launcher verbs, and data preservation by default on uninstall.
- [ ] Run the package test; expect failure because installer files are absent.
- [ ] Implement an Inno wizard for non-secret ports/local LLM only; delegate secret generation to `agentopsctl configure`; add Start Menu entries, post-install start, and explicit opt-in data deletion.
- [ ] Run the package test; expect PASS. Commit as `feat: package Windows self-hosted installer`.

### Task 4: Release workflow and operator guide

**Files:** Create `.github/workflows/windows-release.yml`, `docs/windows-self-hosted.md`; modify package validator scripts.

- [ ] Extend the failing release contract: immutable API/Worker/Web images, Windows amd64 CLI, Inno EXE, SHA-256 checksums, SBOM, and Go/frontend/package gates before release upload.
- [ ] Run the package test; expect failure because workflow/guide are absent.
- [ ] Implement tagged/manual release; pull requests validate but cannot publish. Document Docker Desktop prerequisite, install, local URLs, backup/restore confirmation, update, diagnostics, uninstall, and secret handling.
- [ ] Run `powershell.exe -ExecutionPolicy Bypass -File scripts\validate-windows-package.test.ps1; go test ./...`; expect PASS. GitHub Actions also runs `npm ci && npm test -- --run && npm run build`.
- [ ] Commit as `ci: publish Windows self-hosted installer`.

## Plan review

- Tasks cover runtime, configuration, lifecycle, installer, artifacts, and operator guidance in the approved design.
- Native Windows database services, non-Windows targets, public exposure, cloud deployment, and automated Docker/WSL installation remain excluded.
- Every new behavior gets a direct test or static artifact contract; no lifecycle command accepts arbitrary shell text.
