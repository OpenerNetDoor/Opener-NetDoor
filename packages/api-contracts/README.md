# API Contracts

OpenAPI is source-of-truth for Opener NetDoor admin/user provisioning APIs.

## Files
- `openapi/openapi.v1.yaml` - versioned HTTP contract
- `scripts/validate-openapi.mjs` - lightweight static validation for local development

## Commands

```bash
pnpm --dir packages/api-contracts validate
```

## SDK Generation Seam

Current stage defines generation seams without forcing generator lock-in:
- TypeScript SDK target: `packages/sdk-ts`
- Go SDK target: `packages/sdk-go`

Recommended CI extension (next step):
- spectral lint
- openapi schema validation
- generated SDK drift checks
