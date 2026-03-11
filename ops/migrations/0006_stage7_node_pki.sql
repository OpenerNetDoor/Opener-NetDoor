CREATE TABLE IF NOT EXISTS node_certificates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  serial_number TEXT NOT NULL,
  cert_pem TEXT NOT NULL,
  ca_id TEXT NOT NULL,
  issuer TEXT NOT NULL,
  not_before TIMESTAMPTZ NOT NULL,
  not_after TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  rotate_from_cert_id UUID REFERENCES node_certificates(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_node_certificates_validity_window CHECK (not_after > not_before)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_node_certificates_ca_serial
  ON node_certificates (ca_id, serial_number);

CREATE UNIQUE INDEX IF NOT EXISTS uq_node_certificates_active_per_node
  ON node_certificates (node_id)
  WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_node_certificates_node_created
  ON node_certificates (node_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_node_certificates_tenant_created
  ON node_certificates (tenant_id, created_at DESC);
