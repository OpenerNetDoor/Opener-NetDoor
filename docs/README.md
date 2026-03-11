# Documentation Index

## Core Docs
- `README.md`: monorepo overview and quick start
- `WORKSPACE_POLICY.md`: tooling, lint, CI policy
- `MVP_SCOPE.md`: MVP vs deferred modules

## Architecture and Delivery Stages
- Stage 1–11 decisions are accepted and treated as baseline.
- Stage 12 is implementation-focused and additive.

## API Contracts
- Source-of-truth: `packages/api-contracts/openapi/openapi.v1.yaml`
- Validation script: `packages/api-contracts/scripts/validate-openapi.mjs`

## Runtime Verification Commands
- `go -C services/core-platform test ./...`
- `go -C apps/api-gateway test ./...`
- `powershell -ExecutionPolicy Bypass -File ops/scripts/stage2-integration.ps1`

## Operations
- Production compose: `infra/compose/prod/docker-compose.prod.yml`
- Local compose: `docker-compose.yml`
- Migrations: `ops/migrations`
