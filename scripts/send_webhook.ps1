# scripts/send_webhook.ps1
# Automates HMAC-SHA256 signing and sends synthetic GitHub PR webhook (TASK-MST-303)

$ErrorActionPreference = "Stop"

$Secret = "dev_webhook_secret_12345"
$PayloadPath = Join-Path $PSScriptRoot "payloads\test_pr.json"
$Url = "http://localhost:8081/webhook"

if (-not (Test-Path $PayloadPath)) {
    Write-Error "Payload file not found at: $PayloadPath"
}

# 1. Read exact raw bytes from file
$Bytes = [System.IO.File]::ReadAllBytes($PayloadPath)

# 2. Compute HMAC-SHA256 digest
$Hmac = New-Object System.Security.Cryptography.HMACSHA256
$Hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($Secret)
$HashBytes = $Hmac.ComputeHash($Bytes)
$HexHash = [System.BitConverter]::ToString($HashBytes).Replace("-", "").ToLower()
$SignatureHeader = "sha256=$HexHash"

Write-Host "Computed HMAC-SHA256: $SignatureHeader" -ForegroundColor Cyan
Write-Host "Sending POST request to $Url..." -ForegroundColor Yellow

# 3. Send HTTP POST request with exact payload bytes
$Headers = @{
    "Content-Type"          = "application/json"
    "X-GitHub-Event"        = "pull_request"
    "X-GitHub-Delivery"     = [System.Guid]::NewGuid().ToString()
    "X-Hub-Signature-256"   = $SignatureHeader
}

try {
    $Response = Invoke-WebRequest -Uri $Url -Method Post -Headers $Headers -Body $Bytes
    Write-Host "`n=== SUCCESS: Server Responded ===" -ForegroundColor Green
    Write-Host "Status Code: $($Response.StatusCode)" -ForegroundColor Green
    Write-Host "Response Body: $($Response.Content)" -ForegroundColor Green
} catch {
    Write-Error "Request failed: $_"
}