CREATE TABLE IF NOT EXISTS node_request_nonces (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  node_key_id TEXT NOT NULL,
  request_type TEXT NOT NULL,
  nonce TEXT NOT NULL,
  signed_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT chk_node_request_nonces_request_type CHECK (request_type IN ('register', 'heartbeat')),
  CONSTRAINT chk_node_request_nonces_nonce_length CHECK (char_length(trim(nonce)) >= 8)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_node_request_nonces_identity_nonce
  ON node_request_nonces (tenant_id, node_key_id, request_type, nonce);

CREATE INDEX IF NOT EXISTS idx_node_request_nonces_expires_at
  ON node_request_nonces (expires_at ASC);

CREATE INDEX IF NOT EXISTS idx_node_request_nonces_tenant_consumed
  ON node_request_nonces (tenant_id, consumed_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_time
  ON audit_logs (action, created_at DESC);
