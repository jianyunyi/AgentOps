$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$installer = Join-Path $root 'installer/AgentOps.iss'
if (-not (Test-Path $installer)) { throw 'Windows installer script is required' }
$content = Get-Content -Raw $installer
foreach ($required in @('ArchitecturesAllowed=x64', 'docker compose version', 'wsl.exe', '{autopf}\AgentOps', '{commonappdata}\AgentOps', 'agentops-start.cmd', 'agentops-stop.cmd')) {
  if ($content -notlike "*$required*") { throw "Installer must contain $required" }
}
if ($content -match '(?i)delete.*commonappdata') { throw 'Uninstall must preserve data by default' }
& (Join-Path $PSScriptRoot 'validate-windows-package.ps1') -RepositoryRoot $root
if (-not $?) { throw 'Windows package validation failed' }
Write-Host 'Windows package validation tests passed.'
