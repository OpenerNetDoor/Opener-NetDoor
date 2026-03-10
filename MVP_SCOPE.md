# MVP Scope and Deferred Modules

## In MVP (must be implementation-ready)
- apps/api-gateway
- services/core-platform
- apps/node-agent
- apps/installer-cli
- apps/admin-web (partial scaffold)
- apps/manager-desktop (partial scaffold)
- apps/desktop-client (partial scaffold)
- apps/mobile-client (partial scaffold)
- packages/api-contracts
- ops/migrations
- infra/compose/prod

## Deferred (excluded from MVP runtime, but represented in tree as compile-safe stubs)
- services/auth-service
- services/tenant-service
- services/user-service
- services/key-service
- services/subscription-service
- services/provisioning-service
- services/node-registry-service
- services/stats-service
- services/audit-service
- services/notification-service
- services/usage-worker
- services/webhook-worker

Reason: MVP uses `services/core-platform` as a modular monolith. Deferred services are reserved for decomposition in V2+.
