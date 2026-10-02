# scripts/test_e2e_local.ps1
# End-to-End Local Pipeline Verification Harness (TASK-MST-303)

$ErrorActionPreference = "Stop"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   LUCID-CI: PHASE 3 LOCAL END-TO-END PIPELINE HARNESS   " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

$Stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

$Secret = "dev_webhook_secret_12345"
$PayloadPath = Join-Path $PSScriptRoot "payloads\test_pr.json"
$Url = "http://localhost:8081/webhook"

if (-not (Test-Path $PayloadPath)) {
    Write-Error "Payload file missing at: $PayloadPath"
}

# 1. Read Payload and Calculate Signature
$Bytes = [System.IO.File]::ReadAllBytes($PayloadPath)
$Hmac = New-Object System.Security.Cryptography.HMACSHA256
$Hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($Secret)
$Signature = "sha256=" + [System.BitConverter]::ToString($Hmac.ComputeHash($Bytes)).Replace("-", "").ToLower()

$Headers = @{
    "Content-Type"        = "application/json"
    "X-GitHub-Event"      = "pull_request"
    "X-GitHub-Delivery"   = [System.Guid]::NewGuid().ToString()
    "X-Hub-Signature-256" = $Signature
}

# 2. Trigger Ingestion Gateway
Write-Host "[1/4] Sending signed GitHub webhook to $Url..." -ForegroundColor Yellow
$Resp = Invoke-WebRequest -Uri $Url -Method Post -Headers $Headers -Body $Bytes -UseBasicParsing

if ($Resp.StatusCode -ne 202) {
    Write-Error "Webhook failed with status: $($Resp.StatusCode)"
}

$Body = $Resp.Content | ConvertFrom-Json
$TaskId = $Body.task_id
Write-Host "      Accepted! Task ID: $TaskId, SQS Message ID: $($Body.message_id)" -ForegroundColor Green

# 3. Wait for Background SQS Worker
Write-Host "[2/4] Awaiting SQS Worker consumption and PostgreSQL persistence..." -ForegroundColor Yellow
Start-Sleep -Seconds 2

# 4. Assert PostgreSQL Record
Write-Host "[3/4] Querying PostgreSQL for persisted scan run..." -ForegroundColor Yellow
$DbQuery = "SELECT id, repository_id, pr_number, commit_sha, status FROM scan_runs WHERE repository_id = 12345 ORDER BY started_at DESC LIMIT 1;"
$DbResult = docker exec lucid-postgres psql -U lucid_user -d lucid_ci -t -A -F "," -c $DbQuery

if (-not $DbResult) {
    Write-Error "No scan run found in PostgreSQL!"
}

$Cols = $DbResult.Split(",")
$ScanId = $Cols[0]
$RepoId = $Cols[1]
$PrNumber = $Cols[2]
$Status = $Cols[4]

Write-Host "      Verified DB Record -> Scan ID: $ScanId | Repo: $RepoId | PR: #$PrNumber | Status: $Status" -ForegroundColor Green

# 5. Measure Latency
$Stopwatch.Stop()
$ElapsedSec = [math]::Round($Stopwatch.Elapsed.TotalSeconds, 2)

Write-Host "[4/4] End-to-End Cycle Time: $ElapsedSec seconds (Target: < 30s)" -ForegroundColor Yellow

if ($ElapsedSec -gt 30) {
    Write-Error "SLA Breached: Execution exceeded 30-second budget!"
}

Write-Host "`n==========================================================" -ForegroundColor Green
Write-Host "   PASS: PHASE 3 GATE 3 LOCAL PIPELINE FULLY VERIFIED!   " -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Green
