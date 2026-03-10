CREATE TABLE IF NOT EXISTS tenant_policies (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  traffic_quota_bytes BIGINT,
  device_limit INTEGER,
  default_key_ttl_seconds INTEGER,
  updated_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_tenant_policies_traffic_quota_non_negative CHECK (traffic_quota_bytes IS NULL OR traffic_quota_bytes >= 0),
  CONSTRAINT chk_tenant_policies_device_limit_positive CHECK (device_limit IS NULL OR device_limit > 0),
  CONSTRAINT chk_tenant_policies_default_ttl_positive CHECK (default_key_ttl_seconds IS NULL OR default_key_ttl_seconds > 0)
);

CREATE TABLE IF NOT EXISTS user_policy_overrides (
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  traffic_quota_bytes BIGINT,
  device_limit INTEGER,
  key_ttl_seconds INTEGER,
  updated_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, user_id),
  CONSTRAINT chk_user_policy_overrides_traffic_quota_non_negative CHECK (traffic_quota_bytes IS NULL OR traffic_quota_bytes >= 0),
  CONSTRAINT chk_user_policy_overrides_device_limit_positive CHECK (device_limit IS NULL OR device_limit > 0),
  CONSTRAINT chk_user_policy_overrides_ttl_positive CHECK (key_ttl_seconds IS NULL OR key_ttl_seconds > 0)
);

CREATE INDEX IF NOT EXISTS idx_user_policy_overrides_tenant_updated_at ON user_policy_overrides(tenant_id, updated_at DESC);
