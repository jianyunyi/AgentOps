param(
  [string]$Target = "http://127.0.0.1:8080/health/live",
  [int]$Requests = 1000,
  [int]$Concurrency = 20
)

$ErrorActionPreference = "Stop"
if (-not (Get-Command hey -ErrorAction SilentlyContinue)) { throw "Install hey (https://github.com/rakyll/hey) before running this test" }
Write-Host "Running $Requests requests at concurrency $Concurrency against $Target"
hey -n $Requests -c $Concurrency $Target
