"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { Card, DataTable, ErrorState, HealthChip, PageTitle, StatCard } from "../../components/ui";
import { useAPIClient, useAdminSession } from "../../lib/api/client";
import type { OpsSnapshot } from "@opener-netdoor/shared-types";
import { formatBytes } from "../../lib/format";

export default function OpsSnapshotPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [snapshot, setSnapshot] = useState<OpsSnapshot | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const load = () => {
    void api
      .opsSnapshot(tenantId || undefined)
      .then((result) => {
        setSnapshot(result);
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  useEffect(() => {
    load();
  }, [tenantId]);

  const cards = useMemo(() => {
    if (!snapshot) {
      return [];
    }
    return [
      { id: "traffic", label: "Traffic 24h", value: formatBytes(snapshot.traffic_bytes_24h), tone: "neutral" as const },
      { id: "active", label: "Active certs", value: String(snapshot.active_certificates), tone: "success" as const },
      {
        id: "expiring",
        label: "Expiring certs 24h",
        value: String(snapshot.expiring_certificates_24h),
        tone: snapshot.expiring_certificates_24h > 0 ? ("warning" as const) : ("success" as const),
      },
      {
        id: "replay",
        label: "Replay rejected",
        value: String(snapshot.replay_rejected_24h),
        tone: snapshot.replay_rejected_24h > 0 ? ("warning" as const) : ("success" as const),
      },
    ];
  }, [snapshot]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle title="Ops Snapshot" subtitle="Aggregated operational counters for current scope." />

        <Card title="Scope selector">
          <div className="row">
            <input
              placeholder="tenant_id (blank for platform where allowed)"
              value={tenantId}
              onChange={(event) => setTenantId(event.target.value)}
            />
            <button className="nd-btn is-primary" onClick={load} type="button">
              Reload
            </button>
          </div>
        </Card>

        {error ? <ErrorState message={error} /> : null}

        {snapshot ? (
          <>
            <section className="nd-stat-grid">
              {cards.map((card) => (
                <StatCard key={card.id} label={card.label} value={card.value} tone={card.tone} />
              ))}
            </section>

            <div className="grid-two">
              <Card title="Health chips">
                <HealthChip label="scope" status={snapshot.tenant_id ? "tenant" : "platform"} />
                <HealthChip label="generated_at" status={snapshot.generated_at} />
                <HealthChip label="invalid signature 24h" status={String(snapshot.invalid_signature_24h)} />
              </Card>

              <Card title="Node state counts">
                <DataTable
                  rows={snapshot.node_status}
                  rowKey={(row) => row.status}
                  columns={[
                    { id: "status", header: "Status", render: (row) => row.status },
                    { id: "count", header: "Count", render: (row) => row.count },
                  ]}
                />
              </Card>
            </div>
          </>
        ) : null}
      </AdminShell>
    </RouteGuard>
  );
}