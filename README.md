# Opener NetDoor Platform Monorepo

Production-minded skeleton for a self-hosted, multi-tenant VPN/proxy ecosystem.

## Components

- `apps/api-gateway`: external REST API gateway (JWT auth + scope checks)
- `services/core-platform`: core control-plane service (MVP runtime with PostgreSQL wiring)
- `apps/node-agent`: data-plane node agent
- `apps/installer-cli`: installer and lifecycle CLI
- `apps/admin-web`: web admin panel (partial scaffold)
- `apps/manager-desktop`: desktop manager (partial scaffold)
- `apps/desktop-client`: desktop end-user client (partial scaffold)
- `apps/mobile-client`: mobile client (partial scaffold)
- `services/*`: deferred service decomposition stubs for V2+
- `packages/api-contracts`: OpenAPI contracts
- `ops/migrations`: PostgreSQL migrations
- `infra/compose/prod`: production compose profile
- `infra/compose/test`: stage2+stage3 integration test compose profile

## Quick start (MVP runtime)

```bash
cp .env.example .env
docker compose up -d --build
curl -fsS http://127.0.0.1:8080/v1/health
```

## Admin vertical slice (current)

- Gateway JWT verification (HS256) with issuer/audience/expiry checks.
- Scope checks:
  - `admin:read` for GET `/v1/admin/tenants`, `/v1/admin/users`, `/v1/admin/access-keys`
  - `admin:write` for POST/DELETE write operations.
- Core-platform PostgreSQL-backed handlers:
  - `GET/POST /internal/v1/tenants`
  - `GET/POST /internal/v1/users`
  - `GET/POST/DELETE /internal/v1/access-keys`
- Pagination/filtering baseline:
  - `limit`, `offset`, `status` for list endpoints
  - `tenant_id` and `user_id` filters where relevant
- Structured JSON errors + request IDs in both runtime services.
- Graceful shutdown in both runtime services.

## Stage2+Stage3+Stage4 execution-grade integration checks

- Real PostgreSQL lifecycle for tests via `infra/compose/test/docker-compose.test.yml`
- Migration application verification against `ops/migrations/*.sql`
- Repository + service + HTTP integration tests under build tag `integration`

Run:

```bash
make test-go
make test-stage2
```

On Windows PowerShell:

```powershell
$env:GOCACHE="$PWD\tmpcache\gobuild"
$env:GOMODCACHE="$PWD\tmpcache\gomod"
go -C apps/api-gateway test ./...
go -C services/core-platform test ./...
powershell -ExecutionPolicy Bypass -File ops/scripts/stage2-integration.ps1
```

## Dev token note

Use any JWT tooling to issue a HS256 token with claims:
- `sub` string
- `iss` = `JWT_ISSUER`
- `aud` = `JWT_AUDIENCE`
- `exp` unix timestamp in future
- `scopes` string array containing needed scopes
- `tenant_id` optional tenant-scoped actor binding

## Partial scaffolds

- `apps/admin-web`: compile-ready after dependency install, no business flows yet.
- `apps/manager-desktop`: UI and Tauri shell only.
- `apps/desktop-client`: UI and Tauri shell only.
- `apps/mobile-client`: Flutter shell only.

## Scope control

See `MVP_SCOPE.md` for explicit deferred modules excluded from MVP runtime.
