$ErrorActionPreference = "Stop"

$baseUrl = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8080" }
$token = $env:SMOKE_JWT

if ([string]::IsNullOrWhiteSpace($token)) {
  Write-Error "SMOKE_JWT is required"
}

Write-Host "[stage1] GET $baseUrl/v1/health"
$health = Invoke-WebRequest -Uri "$baseUrl/v1/health" -UseBasicParsing
if ($health.StatusCode -ne 200) {
  Write-Error "health check failed: $($health.StatusCode)"
}

Write-Host "[stage1] GET $baseUrl/v1/ready"
$ready = Invoke-WebRequest -Uri "$baseUrl/v1/ready" -UseBasicParsing
if ($ready.StatusCode -ne 200) {
  Write-Error "ready check failed: $($ready.StatusCode)"
}

Write-Host "[stage1] tenant isolation deny check"
try {
  Invoke-WebRequest -Uri "$baseUrl/v1/admin/users?tenant_id=tenant-deny" -Headers @{ Authorization = "Bearer $token" } -UseBasicParsing | Out-Null
  Write-Error "expected 403 deny path but request succeeded"
} catch {
  $status = $_.Exception.Response.StatusCode.value__
  if ($status -ne 403) {
    Write-Error "expected 403 for tenant isolation deny path, got $status"
  }
}

Write-Host "[stage1] smoke passed"
