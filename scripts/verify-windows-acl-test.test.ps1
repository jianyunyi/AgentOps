$ErrorActionPreference = 'Stop'
$verifier = Join-Path $PSScriptRoot 'verify-windows-acl-test.ps1'
$results = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
$package = 'agentscope/internal/selfhost'
$test = 'TestRestrictWindowsDACL'
$run = @{ Action = 'run'; Package = $package; Test = $test }
$pass = @{ Action = 'pass'; Package = $package; Test = $test }
$packagePass = @{ Action = 'pass'; Package = $package }
$cases = @(
  @{ Name = 'executed and passed'; Events = @($run, $pass, $packagePass); Accepted = $true },
  @{ Name = 'missing test'; Events = @($packagePass); Accepted = $false },
  @{ Name = 'name only in output'; Events = @(@{ Action = 'output'; Package = $package; Output = $test }, $packagePass); Accepted = $false },
  @{ Name = 'skipped'; Events = @($run, @{ Action = 'skip'; Package = $package; Test = $test }, $packagePass); Accepted = $false },
  @{ Name = 'failed'; Events = @($run, @{ Action = 'fail'; Package = $package; Test = $test }, @{ Action = 'fail'; Package = $package }); Accepted = $false },
  @{ Name = 'run without pass'; Events = @($run, $packagePass); Accepted = $false },
  @{ Name = 'pass without run'; Events = @($pass, $packagePass); Accepted = $false },
  @{ Name = 'wrong package'; Events = @(@{ Action = 'run'; Package = 'other'; Test = $test }, @{ Action = 'pass'; Package = 'other'; Test = $test }, $packagePass); Accepted = $false },
  @{ Name = 'package incomplete'; Events = @($run, $pass); Accepted = $false },
  @{ Name = 'another package failed'; Events = @($run, $pass, $packagePass, @{ Action = 'fail'; Package = 'agentscope/cmd/agentopsctl' }); Accepted = $false }
)
try {
  foreach ($case in $cases) {
    $case.Events | ForEach-Object { $_ | ConvertTo-Json -Compress } | Set-Content -LiteralPath $results
    $accepted = $true
    try { & $verifier -ResultsPath $results } catch { $accepted = $false }
    if ($accepted -ne $case.Accepted) { throw "Unexpected verification result: $($case.Name)" }
    Write-Host "PASS: $($case.Name)"
  }
} finally {
  Remove-Item -LiteralPath $results -ErrorAction SilentlyContinue
}
