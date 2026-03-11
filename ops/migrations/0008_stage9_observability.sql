-- Stage 9: observability and operations query indexes

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created_at
  ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created_at
  ON audit_logs (action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_nodes_tenant_status
  ON nodes (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_node_certificates_tenant_not_after_active
  ON node_certificates (tenant_id, not_after)
  WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_traffic_usage_hourly_tenant_ts_hour
  ON traffic_usage_hourly (tenant_id, ts_hour DESC);
