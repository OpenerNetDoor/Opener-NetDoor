"use client";

import { FormEvent, useEffect, useState } from "react";
import type { EffectivePolicy } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, ErrorState, PageTitle, StatusBadge } from "../../../components/ui";
import { useAPIClient, useAdminSession } from "../../../lib/api/client";

export default function EffectivePolicyPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [userId, setUserId] = useState("");
  const [policy, setPolicy] = useState<EffectivePolicy | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const resolve = (event: FormEvent) => {
    event.preventDefault();
    if (!tenantId.trim() || !userId.trim()) {
      setError("tenant_id and user_id are required");
      return;
    }
    void api
      .getEffectivePolicy(tenantId.trim(), userId.trim())
      .then((result) => {
        setPolicy(result);
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle title="Effective Policy" subtitle="Merged policy resolution for tenant default + user override." />
        <Card title="Resolve">
          <form onSubmit={resolve} className="row">
            <input placeholder="tenant_id" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <input placeholder="user_id" value={userId} onChange={(event) => setUserId(event.target.value)} />
            <button type="submit">Resolve</button>
          </form>
        </Card>

        {error ? <ErrorState message={error} /> : null}

        {policy ? (
          <Card title="Policy result">
            <div className="row">
              <p>
                source <StatusBadge value={policy.source} />
              </p>
              <p>
                quota exceeded <StatusBadge value={policy.quota_exceeded ? "yes" : "no"} />
              </p>
            </div>
            <pre>{JSON.stringify(policy, null, 2)}</pre>
          </Card>
        ) : null}
      </AdminShell>
    </RouteGuard>
  );
}
