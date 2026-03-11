CREATE TABLE IF NOT EXISTS pki_issuers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  issuer_id TEXT NOT NULL UNIQUE,
  source TEXT NOT NULL,
  ca_id TEXT NOT NULL UNIQUE,
  issuer_name TEXT NOT NULL,
  ca_cert_pem TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  activated_at TIMESTAMPTZ,
  retired_at TIMESTAMPTZ,
  rotate_from_issuer_id TEXT REFERENCES pki_issuers(issuer_id) ON DELETE SET NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_pki_issuers_source CHECK (source IN ('file', 'external')),
  CONSTRAINT chk_pki_issuers_status CHECK (status IN ('pending', 'active', 'retired')),
  CONSTRAINT chk_pki_issuers_status_timestamps CHECK (
    (status = 'active' AND activated_at IS NOT NULL)
    OR (status = 'pending' AND activated_at IS NULL)
    OR (status = 'retired' AND retired_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pki_issuers_single_active
  ON pki_issuers ((status))
  WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_pki_issuers_status_created
  ON pki_issuers (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pki_issuers_retired_at
  ON pki_issuers (retired_at DESC);

ALTER TABLE node_certificates
  ADD COLUMN IF NOT EXISTS issuer_id TEXT;

CREATE INDEX IF NOT EXISTS idx_node_certificates_issuer_id
  ON node_certificates (issuer_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_node_certificates_issuer_id'
  ) THEN
    ALTER TABLE node_certificates
      ADD CONSTRAINT fk_node_certificates_issuer_id
      FOREIGN KEY (issuer_id)
      REFERENCES pki_issuers(issuer_id)
      ON DELETE SET NULL;
  END IF;
END $$;

UPDATE node_certificates nc
SET issuer_id = pi.issuer_id
FROM pki_issuers pi
WHERE nc.issuer_id IS NULL
  AND nc.ca_id = pi.ca_id;
