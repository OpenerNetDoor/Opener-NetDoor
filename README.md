# Opener NetDoor Monorepo

Opener NetDoor is a self-hosted, multi-tenant VPN/proxy control-plane platform.

This repository contains:
- Production control-plane runtime (`services/core-platform`, `apps/api-gateway`)
- Installer and node agent foundations (`apps/installer-cli`, `apps/node-agent`)
- Admin and manager app skeletons (`apps/admin-web`, `apps/manager-desktop`)
- Mobile client skeleton (`apps/mobile-client`)
- API contracts and SDK seams (`packages/api-contracts`, `packages/sdk-go`, `packages/sdk-ts`, `packages/shared-types`)

## Stage Status

Stages 1–11 are accepted.
Stage 12 is active and focused on compile-oriented skeleton hardening and delivery seams.

## Monorepo Layout

- `apps/api-gateway`: external REST gateway, JWT auth, tenant isolation
- `services/core-platform`: modular monolith control-plane runtime
- `apps/installer-cli`: install/upgrade/rollback/backup/restore/doctor CLI
- `apps/node-agent`: node registration/heartbeat/provisioning seam
- `apps/admin-web`: admin panel routes for operations
- `apps/manager-desktop`: Tauri manager skeleton
- `apps/mobile-client`: Flutter client skeleton
- `packages/api-contracts`: OpenAPI source-of-truth and validation scripts
- `packages/sdk-go`: Go client SDK seams for ops/bootstrap
- `packages/sdk-ts`: TypeScript SDK seams for admin/client apps
- `packages/shared-types`: shared TypeScript DTO primitives
- `ops/migrations`: PostgreSQL migrations
- `infra/compose/prod`: production compose profile
- `infra/compose/test`: test/integration compose profile
- `deploy`: single-owner VPS deployment flow (compose + install/upgrade/uninstall)

## Quick Start (Local Runtime)

```bash
cp .env.example .env
docker compose up -d --build
curl -fsS http://127.0.0.1:8080/v1/health
```

## Self-Hosted VPS Install

```bash
bash deploy/install.sh --ip-mode
```

Domain + HTTPS mode:

```bash
bash deploy/install.sh --domain panel.example.com --email ops@example.com
```

See [deploy/README.md](deploy/README.md) for full deployment details.

## Core Commands

```bash
make test-go
make test-stage2
make validate-contracts
make dev-installer-help
```

## Workspace Policy

Read:
- `WORKSPACE_POLICY.md`
- `MVP_SCOPE.md`
- `docs/README.md`

## Notes

- OpenAPI in `packages/api-contracts/openapi/openapi.v1.yaml` is source-of-truth.
- Stage 12 adds additive skeletons only and does not rewrite Stage 1–11 runtime contracts.
