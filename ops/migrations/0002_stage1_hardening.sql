-- Stage 1 hardening constraints for conflict/error mapping.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_name ON tenants(name);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_email ON users(tenant_id, email) WHERE email IS NOT NULL AND email <> '';
