param([Parameter(Mandatory)][string]$RepositoryRoot)
$ErrorActionPreference = 'Stop'
$compose = Get-Content -Raw (Join-Path $RepositoryRoot 'deploy/windows/compose.yaml')
$example = Get-Content -Raw (Join-Path $RepositoryRoot 'deploy/windows/agentops.env.example')
foreach ($service in @('web:', 'api:', 'worker:', 'mysql:', 'redis:')) { if ($compose -notmatch "(?m)^  $service") { throw "missing service $service" } }
if ($compose -notmatch '127\.0\.0\.1:\$\{WEB_PORT\}:3000' -or $compose -notmatch '127\.0\.0\.1:\$\{API_PORT\}:8080') { throw 'host ports must be loopback only' }
if (($compose | Select-String -AllMatches 'healthcheck').Matches.Count -lt 5) { throw 'all services require health checks' }
if ($compose -match ':latest\b') { throw 'floating image tag is forbidden' }
if ($example -match '(?m)^(MYSQL_PASSWORD|SESSION_SECRET|AGENT_SIGNING_ENCRYPTION_KEY)=.+' ) { throw 'example must not contain secret values' }
