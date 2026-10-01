# AgentOps Windows self-hosted installation

## Prerequisites

Use Windows 10/11 x64 with WSL 2 and Docker Desktop installed and running. Download the `AgentOps-Setup-x64.exe` artifact and its `SHA256SUMS` file from the matching GitHub Release; verify the checksum before starting the installer.

## Install and first start

Run the installer as an administrator. It validates WSL 2 and `docker compose`, installs runtime files in `%ProgramFiles%\AgentOps`, and keeps configuration/data in `%ProgramData%\AgentOps`. The installer requires accessible API/Web image references pinned by SHA-256 digest, and generates the database password, local session and agent-signing secrets. Configuration errors fail installation; reruns preserve valid existing configuration. Do not copy those files into source control or support tickets.

Use the Start Menu entries to start AgentOps, stop it, view logs, or run diagnostics. After the API readiness check completes, open `http://127.0.0.1:3300`. API health is `http://127.0.0.1:8080/health/ready`.

## Local LLM

The first release leaves local LLM disabled unless explicitly configured. Enable it only when an Ollama-compatible runtime and model are available; retain the default loopback-only network exposure.

## Backup and restore

Run recovery commands in an elevated PowerShell. Backups contain sensitive database data and the original configuration/key material, encrypted using AES-256-GCM with a random salt and scrypt-derived key. Keep a strong backup password in a separate password manager; losing it makes recovery impossible. This version explicitly limits the uncompressed SQL plus configuration to 256 MiB.

Stop AgentOps first (`agentopsctl stop`). Backup verifies that API, worker and web are stopped, starts only MySQL/Redis if needed, and uses an authenticated, transactional MySQL export. Prohibit external writers and schema changes throughout recovery operations. It publishes a randomly named `.aob` file in `%ProgramData%\AgentOps\backups` only after successful export, sync and authentication/checksum verification. Failures remove pending files. Windows configuration and backup DACLs allow only Administrators and SYSTEM.

```powershell
# Obtain the password through your password manager; do not put it in shell history.
$credential = Read-Host 'Backup password (at least 16 characters)' -AsSecureString
$env:AGENTOPS_BACKUP_PASSWORD = [System.Net.NetworkCredential]::new('', $credential).Password
& "$env:ProgramFiles\AgentOps\agentopsctl.exe" stop
& "$env:ProgramFiles\AgentOps\agentopsctl.exe" backup
& "$env:ProgramFiles\AgentOps\agentopsctl.exe" verify-backup 'C:\private\agentops-backup.aob'
Remove-Item Env:AGENTOPS_BACKUP_PASSWORD
```

Restore into a dedicated clean installation with fresh Docker volumes, never the live production project. On a new host, preserve then remove the newly generated `agentops.env` from its protected config directory before restore; the authenticated backup supplies the original configuration, database password and signing-encryption key. If a config already exists, restore requires an exact byte match and never overwrites it. Original digest-pinned images must still be accessible.

```powershell
# Set AGENTOPS_BACKUP_PASSWORD securely as above.
& "$env:ProgramFiles\AgentOps\agentopsctl.exe" restore 'C:\private\agentops-backup.aob' --confirm
& "$env:ProgramFiles\AgentOps\agentopsctl.exe" start
Remove-Item Env:AGENTOPS_BACKUP_PASSWORD
```

Restore authenticates the bundle and validates its manifest, checksums and configuration before import. It refuses running application services or a destination database with any tables or a nonempty Redis. Recovery operations on the same configuration are serialized by `.recovery-lock`; after a process crash, inspect for active recovery processes before removing a stale lock. Import failure leaves a partial destination: discard that test destination and retry against a fresh database; do not continue using it. Restore does not delete source volumes or automatically start the application.

Redis is deliberately **not** backed up: fresh-instance restore resets sessions, HMAC replay nonces and rate-limit counters. Rotate Agent credentials/revoke old access before exposing the recovered system; wait out the configured replay window before reopening ingest. MySQL events, users and encrypted Agent credentials are retained. After restore, verify login, credential decryption, a newly signed ingest request and event counts before accepting traffic.

CI runs a real MySQL 8.4 export/import round trip and verifies recovery of encrypted credential data with the preserved key, alongside wrong-password, tamper, partial-export and destination-guard tests. A release still requires a Windows Docker Desktop install/recovery drill and application-level login/ingest acceptance.

## Update and uninstall

Use a newer GitHub Release only after verifying its checksum. Stop AgentOps before updating; the installer preserves `%ProgramData%\AgentOps` secrets, backups, and Docker volumes. Uninstall also preserves that data by default. Remove data only after making and testing a backup.

## Diagnostics and support

Run **Diagnose AgentOps** to confirm Docker Desktop and Compose availability. Share container status and redacted logs only; never share `agentops.env`, passwords, session secrets, signing keys, or DSNs.
