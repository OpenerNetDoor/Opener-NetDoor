-- Stage Next: runtime layer for Xray VLESS+REALITY and admin cookie sessions

CREATE TABLE IF NOT EXISTS node_runtimes (
  node_id UUID PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  runtime_backend TEXT NOT NULL DEFAULT 'xray',
  runtime_protocol TEXT NOT NULL DEFAULT 'vless_reality',
  listen_port INTEGER NOT NULL DEFAULT 8443,
  reality_public_key TEXT NOT NULL,
  reality_short_id TEXT NOT NULL,
  reality_server_name TEXT NOT NULL,
  applied_config_version INTEGER NOT NULL DEFAULT 0,
  runtime_status TEXT NOT NULL DEFAULT 'pending',
  last_applied_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_node_runtimes_backend CHECK (runtime_backend IN ('xray')),
  CONSTRAINT chk_node_runtimes_protocol CHECK (runtime_protocol IN ('vless_reality')),
  CONSTRAINT chk_node_runtimes_port CHECK (listen_port >= 1 AND listen_port <= 65535)
);

CREATE INDEX IF NOT EXISTS idx_node_runtimes_tenant_status
  ON node_runtimes (tenant_id, runtime_status);

CREATE TABLE IF NOT EXISTS runtime_revisions (
  id BIGSERIAL PRIMARY KEY,
  node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  config_json JSONB NOT NULL,
  applied BOOLEAN NOT NULL DEFAULT FALSE,
  applied_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(node_id, version)
);

CREATE INDEX IF NOT EXISTS idx_runtime_revisions_node_created
  ON runtime_revisions (node_id, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_sessions (
  session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject TEXT NOT NULL,
  tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
  scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires
  ON admin_sessions (expires_at)
  WHERE revoked_at IS NULL;
