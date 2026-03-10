ALTER TABLE nodes
  ADD COLUMN IF NOT EXISTS node_key_id TEXT,
  ADD COLUMN IF NOT EXISTS node_public_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS contract_version TEXT NOT NULL DEFAULT '2026-03-10.stage5.v1',
  ADD COLUMN IF NOT EXISTS agent_version TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS identity_fingerprint TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ;

UPDATE nodes
SET node_key_id = 'legacy-' || id::text
WHERE COALESCE(node_key_id, '') = '';

ALTER TABLE nodes
  ALTER COLUMN node_key_id SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_nodes_status_stage5'
  ) THEN
    ALTER TABLE nodes
      ADD CONSTRAINT chk_nodes_status_stage5
      CHECK (status IN ('pending', 'active', 'stale', 'offline', 'revoked'));
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_nodes_tenant_node_key ON nodes(tenant_id, node_key_id);
CREATE INDEX IF NOT EXISTS idx_nodes_tenant_status ON nodes(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_nodes_last_heartbeat_at ON nodes(last_heartbeat_at DESC);

CREATE TABLE IF NOT EXISTS node_heartbeats (
  id BIGSERIAL PRIMARY KEY,
  node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_node_heartbeats_node_received ON node_heartbeats(node_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_node_heartbeats_tenant_received ON node_heartbeats(tenant_id, received_at DESC);
