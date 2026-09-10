# AgentOps Windows self-hosted installation

## Prerequisites

Use Windows 10/11 x64 with WSL 2 and Docker Desktop installed and running. Download the `AgentOps-Setup-x64.exe` artifact and its `SHA256SUMS` file from the matching GitHub Release; verify the checksum before starting the installer.

## Install and first start

Run the installer as an administrator. It validates WSL 2 and `docker compose`, installs runtime files in `%ProgramFiles%\AgentOps`, and keeps configuration/data in `%ProgramData%\AgentOps`. The first configuration generates local session and agent-signing secrets. Do not copy those files into source control or support tickets.

Use the Start Menu entries to start AgentOps, stop it, view logs, or run diagnostics. After the API readiness check completes, open `http://127.0.0.1:3300`. API health is `http://127.0.0.1:8080/health/ready`.

## Local LLM

The first release leaves local LLM disabled unless explicitly configured. Enable it only when an Ollama-compatible runtime and model are available; retain the default loopback-only network exposure.

## Backup and restore

Run the **Backup** command through `agentopsctl backup` after stopping write-heavy work. SQL exports are placed in `%ProgramData%\AgentOps\backups`. Restore is intentionally manual and destructive: stop AgentOps, create a fresh backup, then use a reviewed MySQL restore command for the selected SQL file. Never overwrite a volume without a known-good backup.

## Update and uninstall

Use a newer signed GitHub Release only after verifying its checksum. Stop AgentOps before updating; the installer preserves `%ProgramData%\AgentOps` secrets, backups, and Docker volumes. Uninstall also preserves that data by default. Remove data only after making and testing a backup.

## Diagnostics and support

Run **Diagnose AgentOps** to confirm Docker Desktop and Compose availability. Share container status and redacted logs only; never share `agentops.env`, passwords, session secrets, signing keys, or DSNs.
