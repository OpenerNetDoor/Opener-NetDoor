"use client";

import { useParams, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { Node, NodeCertificate } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, DataTable, ErrorState, LoadingState, PageTitle, StatusBadge } from "../../../components/ui";
import { useAPIClient, useOwnerScopeId } from "../../../lib/api/client";

export default function NodeDetailPage() {
  const params = useParams<{ nodeId: string }>();
  const search = useSearchParams();
  const api = useAPIClient();
  const ownerScopeId = useOwnerScopeId();

  const nodeId = useMemo(() => String(params.nodeId ?? ""), [params]);
  const tenantId = search.get("tenant_id") ?? ownerScopeId ?? "";

  const [node, setNode] = useState<Node | null>(null);
  const [certs, setCerts] = useState<NodeCertificate[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId.trim() || !nodeId) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const nodes = await api.listNodesPage({ tenantId, limit: 100, offset: 0 });
        const found = nodes.items.find((item) => item.id === nodeId) ?? null;
        const certList = await api.listNodeCertificatesPage({ tenantId, nodeId, limit: 50, offset: 0 });
        if (cancelled) {
          return;
        }
        setNode(found);
        setCerts(certList.items);
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
  }, [api, tenantId, nodeId]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle title="Server Detail" subtitle="Diagnostics and certificate history." />

        <Card title="Context">
          <p>
            server id <code>{nodeId}</code>
          </p>
          <p>
            workspace scope <code>{tenantId || "n/a"}</code>
          </p>
        </Card>

        {error ? <ErrorState message={error} /> : null}
        {!node && !error ? <LoadingState label="Loading server..." /> : null}

        {node ? (
          <Card title="Server">
            <p>
              host <strong>{node.hostname}</strong> / location <strong>{node.region}</strong>
            </p>
            <p>
              status <StatusBadge value={node.status} />
            </p>
            <p>last heartbeat {node.last_heartbeat_at ?? "n/a"}</p>
            <details>
              <summary>Advanced internals</summary>
              <p>
                node key <code>{node.node_key_id}</code>
              </p>
              <p>
                contract {node.contract_version || "n/a"}, agent {node.agent_version || "n/a"}
              </p>
            </details>
          </Card>
        ) : null}

        <Card title="Certificate history">
          <DataTable
            rows={certs}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "Certificate", render: (row) => <code>{row.id}</code> },
              { id: "serial", header: "Serial", render: (row) => <code>{row.serial_number}</code> },
              {
                id: "status",
                header: "Status",
                render: (row) => <StatusBadge value={row.revoked_at ? "revoked" : "active"} />,
              },
              { id: "issuer", header: "Issuer", render: (row) => row.issuer || row.issuer_id || "n/a" },
              { id: "not_after", header: "Not after", render: (row) => row.not_after },
            ]}
          />
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
