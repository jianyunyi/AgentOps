param(
  [Parameter(Mandatory = $true)]
  [string]$ResultsPath
)

$ErrorActionPreference = 'Stop'
$events = @(Get-Content -LiteralPath $ResultsPath | ForEach-Object { $_ | ConvertFrom-Json })
$package = 'agentscope/internal/selfhost'
$test = 'TestRestrictWindowsDACL'
$testEvents = @($events | Where-Object { $_.Package -eq $package -and $_.Test -eq $test })
if (-not ($testEvents | Where-Object { $_.Action -eq 'run' }) -or
    -not ($testEvents | Where-Object { $_.Action -eq 'pass' }) -or
    ($testEvents | Where-Object { $_.Action -in @('skip', 'fail') })) {
  throw 'Windows ACL verification must execute and pass without skipping'
}
$packageEvents = @($events | Where-Object { $_.Package -eq $package -and -not $_.Test })
if (-not ($packageEvents | Where-Object { $_.Action -eq 'pass' }) -or
    ($events | Where-Object { $_.Action -eq 'fail' })) {
  throw 'Windows control test packages must pass'
}
Write-Host 'Windows protected DACL test executed and passed.'
