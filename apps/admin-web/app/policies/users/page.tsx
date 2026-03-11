"use client";

import { FormEvent, useEffect, useState } from "react";
import type { UserPolicyOverride } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, DataTable, ErrorState, PageTitle, PaginationControls } from "../../../components/ui";
import { useAPIClient, useAdminSession } from "../../../lib/api/client";
import { hasScopes } from "../../../lib/permissions";

export default function UserOverridesPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [userIdFilter, setUserIdFilter] = useState("");
  const [items, setItems] = useState<UserPolicyOverride[]>([]);
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [error, setError] = useState("");

  const [upsertUserId, setUpsertUserId] = useState("");
  const [quotaBytes, setQuotaBytes] = useState("");
  const [deviceLimit, setDeviceLimit] = useState("");
  const [keyTTL, setKeyTTL] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const load = () => {
    if (!tenantId.trim()) {
      setError("tenant_id is required");
      return;
    }
    void api
      .listUserPolicyOverridesPage({
        tenantId: tenantId.trim(),
        userId: userIdFilter.trim() || undefined,
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
    if (tenantId.trim()) {
      load();
    }
  }, [limit, offset]);

  const upsert = (event: FormEvent) => {
    event.preventDefault();
    if (!tenantId.trim() || !upsertUserId.trim()) {
      setError("tenant_id and user_id are required");
      return;
    }
    void api
      .upsertUserPolicyOverride({
        tenant_id: tenantId.trim(),
        user_id: upsertUserId.trim(),
        traffic_quota_bytes: quotaBytes.trim() ? Number(quotaBytes) : null,
        device_limit: deviceLimit.trim() ? Number(deviceLimit) : null,
        key_ttl_seconds: keyTTL.trim() ? Number(keyTTL) : null,
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
        <PageTitle title="User Policy Overrides" subtitle="Optional user-level overrides on top of tenant defaults." />

        <Card title="Filter" actions={<button onClick={load}>Load</button>}>
          <div className="row">
            <input placeholder="tenant_id" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <input placeholder="user_id (optional)" value={userIdFilter} onChange={(event) => setUserIdFilter(event.target.value)} />
            <button onClick={load}>Apply</button>
          </div>
        </Card>

        {canWrite ? (
          <Card title="Upsert user override">
            <form onSubmit={upsert} className="row">
              <input placeholder="user_id" value={upsertUserId} onChange={(event) => setUpsertUserId(event.target.value)} />
              <input placeholder="traffic_quota_bytes" value={quotaBytes} onChange={(event) => setQuotaBytes(event.target.value)} />
              <input placeholder="device_limit" value={deviceLimit} onChange={(event) => setDeviceLimit(event.target.value)} />
              <input placeholder="key_ttl_seconds" value={keyTTL} onChange={(event) => setKeyTTL(event.target.value)} />
              <button type="submit">Save override</button>
            </form>
          </Card>
        ) : null}

        <Card title="Overrides">
          {error ? <ErrorState message={error} /> : null}
          <DataTable
            rows={items}
            rowKey={(row) => `${row.tenant_id}-${row.user_id}`}
            columns={[
              { id: "tenant", header: "Tenant", render: (row) => <code>{row.tenant_id}</code> },
              { id: "user", header: "User", render: (row) => <code>{row.user_id}</code> },
              { id: "quota", header: "Quota", render: (row) => row.traffic_quota_bytes ?? "none" },
              { id: "devices", header: "Device limit", render: (row) => row.device_limit ?? "none" },
              { id: "ttl", header: "Key TTL", render: (row) => row.key_ttl_seconds ?? "none" },
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
