"use client";

import { FormEvent, useEffect, useState } from "react";
import type { TenantPolicy } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, DataTable, ErrorState, PageTitle, PaginationControls } from "../../../components/ui";
import { useAPIClient, useAdminSession } from "../../../lib/api/client";
import { hasScopes } from "../../../lib/permissions";

export default function TenantPoliciesPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [items, setItems] = useState<TenantPolicy[]>([]);
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [error, setError] = useState("");

  const [quotaBytes, setQuotaBytes] = useState("");
  const [deviceLimit, setDeviceLimit] = useState("");
  const [defaultTTL, setDefaultTTL] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const load = () => {
    void api
      .listTenantPoliciesPage({
        tenantId: tenantId || undefined,
        limit,
        offset,
      })
      .then((result) => {
        setItems(result.items);
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  useEffect(() => {
    load();
  }, [limit, offset]);

  const upsert = (event: FormEvent) => {
    event.preventDefault();
    if (!tenantId.trim()) {
      setError("tenant_id is required");
      return;
    }
    void api
      .upsertTenantPolicy({
        tenant_id: tenantId.trim(),
        traffic_quota_bytes: quotaBytes.trim() ? Number(quotaBytes) : null,
        device_limit: deviceLimit.trim() ? Number(deviceLimit) : null,
        default_key_ttl_seconds: defaultTTL.trim() ? Number(defaultTTL) : null,
      })
      .then(() => {
        setError("");
        load();
      })
      .catch((err) => setError(err instanceof Error ? err.message : "upsert failed"));
  };

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle title="Tenant Policies" subtitle="Default traffic quota, device limits, and key TTL policy per tenant." />

        <Card title="Filter" actions={<button onClick={load}>Refresh</button>}>
          <div className="row">
            <input placeholder="tenant_id (optional)" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <button onClick={load}>Load</button>
          </div>
        </Card>

        {canWrite ? (
          <Card title="Upsert policy">
            <form onSubmit={upsert} className="row">
              <input placeholder="traffic_quota_bytes" value={quotaBytes} onChange={(event) => setQuotaBytes(event.target.value)} />
              <input placeholder="device_limit" value={deviceLimit} onChange={(event) => setDeviceLimit(event.target.value)} />
              <input placeholder="default_key_ttl_seconds" value={defaultTTL} onChange={(event) => setDefaultTTL(event.target.value)} />
              <button type="submit">Save policy</button>
            </form>
          </Card>
        ) : null}

        <Card title="Policies">
          {error ? <ErrorState message={error} /> : null}
          <DataTable
            rows={items}
            rowKey={(row) => row.tenant_id}
            columns={[
              { id: "tenant", header: "Tenant", render: (row) => <code>{row.tenant_id}</code> },
              { id: "quota", header: "Traffic quota", render: (row) => row.traffic_quota_bytes ?? "none" },
              { id: "devices", header: "Device limit", render: (row) => row.device_limit ?? "none" },
              { id: "ttl", header: "Default key TTL", render: (row) => row.default_key_ttl_seconds ?? "none" },
              { id: "updated", header: "Updated", render: (row) => row.updated_at },
            ]}
          />
          <PaginationControls
            limit={limit}
            offset={offset}
            onChange={({ limit: nextLimit, offset: nextOffset }) => {
              setLimit(nextLimit);
              setOffset(nextOffset);
            }}
          />
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
