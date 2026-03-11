"use client";

import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { Tenant, TenantPolicy } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, ErrorState, LoadingState, PageTitle, StatusBadge } from "../../../components/ui";
import { useAPIClient } from "../../../lib/api/client";

export default function TenantDetailPage() {
  const params = useParams<{ tenantId: string }>();
  const tenantId = useMemo(() => String(params.tenantId ?? ""), [params]);
  const api = useAPIClient();

  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [policy, setPolicy] = useState<TenantPolicy | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const tenants = await api.listTenantsPage({ limit: 100, offset: 0 });
        const found = tenants.items.find((item) => item.id === tenantId) ?? null;
        const policyResponse = await api.getTenantPolicy(tenantId);
        if (cancelled) {
          return;
        }
        setTenant(found);
        setPolicy(policyResponse);
        setError("");
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "request failed");
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [api, tenantId]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId}>
      <AdminShell>
        <PageTitle title="Tenant Detail" subtitle={`tenant_id=${tenantId}`} />
        {error ? <ErrorState message={error} /> : null}
        {!tenant && !error ? <LoadingState label="Loading tenant..." /> : null}
        {tenant ? (
          <Card title="Tenant">
            <p>
              <strong>{tenant.name}</strong>
            </p>
            <p>
              status <StatusBadge value={tenant.status} />
            </p>
            <p>
              tenant id <code>{tenant.id}</code>
            </p>
            <p>created {tenant.created_at}</p>
          </Card>
        ) : null}

        <Card title="Tenant policy">
          {policy ? (
            <pre>{JSON.stringify(policy, null, 2)}</pre>
          ) : (
            <p>No explicit tenant policy. Effective defaults from service layer apply.</p>
          )}
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
