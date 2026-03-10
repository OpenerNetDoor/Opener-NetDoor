$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "../..")
$composeFile = Join-Path $root "infra/compose/test/docker-compose.test.yml"
$projectName = "openernetdoor-stage2"
$testPostgresPort = if ($env:TEST_POSTGRES_PORT) { $env:TEST_POSTGRES_PORT } else { "35432" }
$env:TEST_POSTGRES_PORT = $testPostgresPort

function Cleanup {
  docker compose -p $projectName -f $composeFile --profile stage2-test down -v | Out-Null
}

try {
  docker compose -p $projectName -f $composeFile --profile stage2-test up -d | Out-Null

  Write-Host "Waiting for postgres-test to become healthy on port $testPostgresPort..."
  $healthy = $false
  for ($i = 0; $i -lt 40; $i++) {
    $status = docker inspect --format='{{json .State.Health.Status}}' "$projectName-postgres-test-1" 2>$null
    if ($status -eq '"healthy"') {
      $healthy = $true
      break
    }
    Start-Sleep -Seconds 2
  }
  if (-not $healthy) {
    throw "postgres-test did not become healthy"
  }

  $env:TEST_DATABASE_URL = "postgresql://openernetdoor:openernetdoor@127.0.0.1:$testPostgresPort/openernetdoor_test?sslmode=disable"
  $env:TEST_MIGRATIONS_DIR = (Join-Path $root "ops/migrations")
  $env:GOCACHE = (Join-Path $root "tmpcache/gobuild")
  $env:GOMODCACHE = (Join-Path $root "tmpcache/gomod")

  New-Item -ItemType Directory -Force $env:GOCACHE | Out-Null
  New-Item -ItemType Directory -Force $env:GOMODCACHE | Out-Null

  go -C (Join-Path $root "services/core-platform") test -p 1 -tags=integration ./internal/store ./internal/service ./internal/http -count=1 -v
  go -C (Join-Path $root "apps/api-gateway") test -p 1 -tags=integration ./internal/http -count=1 -v
} finally {
  Cleanup
}
