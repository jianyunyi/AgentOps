$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
& (Join-Path $PSScriptRoot 'validate-windows-package.ps1') -RepositoryRoot $root
if (-not $?) { throw 'Windows package validation failed' }
Write-Host 'Windows package validation tests passed.'
