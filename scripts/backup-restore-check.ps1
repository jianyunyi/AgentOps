param(
  [Parameter(Mandatory = $true)][string]$MysqlContainer,
  [Parameter(Mandatory = $true)][string]$MysqlPassword,
  [string]$Database = "agentscope",
  [string]$BackupFile = ".\agentscope-backup.sql"
)

$ErrorActionPreference = "Stop"
$mysql = "mysql"

Write-Host "Creating logical backup: $BackupFile"
docker exec $MysqlContainer mysqldump --single-transaction --routines --triggers -uroot "-p$MysqlPassword" $Database | Out-File -FilePath $BackupFile -Encoding utf8
if (-not (Test-Path -LiteralPath $BackupFile)) { throw "Backup file was not created" }

Write-Host "Backup created. Restore verification requires a disposable database."
Write-Host "Example: docker exec -i $MysqlContainer $mysql -uroot -p<password> -e 'CREATE DATABASE restore_check'"
Write-Host "Then: Get-Content -Raw -LiteralPath $BackupFile | docker exec -i $MysqlContainer $mysql -uroot -p<password> restore_check"
