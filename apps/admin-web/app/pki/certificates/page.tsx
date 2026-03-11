"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import {
  ActionButton,
  Card,
  DataTable,
  ErrorState,
  PageTitle,
  PaginationControls,
  StatusBadge,
  SupportBadge,
} from "../../../components/ui";
import { useAPIClient, useAdminSession } from "../../../lib/api/client";
import { hasScopes } from "../../../lib/permissions";
import { toCertificatesVM, type CertificateVM } from "../../../lib/adapters/pki";

export default function PKICertificatesPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [status, setStatus] = useState<"active" | "revoked" | "">("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<CertificateVM[]>([]);
  const [error, setError] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const load = () => {
    if (!tenantId.trim() || !nodeId.trim()) {
      setError("tenant_id and node_id are required");
      return;
    }

    void api
      .listNodeCertificatesPage({
        tenantId: tenantId.trim(),
        nodeId: nodeId.trim(),
        status: status || undefined,
        limit,
        offset,
      })
      .then((result) => {
        setItems(toCertificatesVM(result.items));
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  useEffect(() => {
    if (tenantId.trim() && nodeId.trim()) {
      load();
    }
  }, [limit, offset]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle
          title="Node Certificates"
          subtitle="Certificate history, expiry status, and rotation controls."
          actions={<SupportBadge state={canWrite ? "supported" : "unsupported"} />}
        />

        <Card title="Lookup context">
          <div className="row">
            <input placeholder="tenant_id" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <input placeholder="node_id" value={nodeId} onChange={(event) => setNodeId(event.target.value)} />
            <select value={status} onChange={(event) => setStatus(event.target.value as "active" | "revoked" | "")}> 
              <option value="">all</option>
              <option value="active">active</option>
              <option value="revoked">revoked</option>
            </select>
            <button className="nd-btn is-primary" onClick={load} type="button">
              Load
            </button>
          </div>
        </Card>

        {error ? <ErrorState message={error} /> : null}

        <Card title="Certificate history" subtitle="Issue endpoint is not exposed as a standalone operation in current backend.">
          <DataTable
            rows={items}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "ID", render: (row) => <code>{row.id}</code> },
              { id: "serial", header: "Serial", render: (row) => <code>{row.serial}</code> },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "issuer", header: "Issuer", render: (row) => row.issuer },
              { id: "not_after", header: "Not after", render: (row) => row.notAfter },
              { id: "rotated", header: "Rotated from", render: (row) => row.rotatedFrom },
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

        {canWrite ? (
          <Card title="Certificate operations" subtitle="Rotate/renew/revoke are backend-supported endpoints.">
            <div className="row">
              <ActionButton
                onClick={() => {
                  if (!tenantId.trim() || !nodeId.trim()) {
                    setError("tenant_id and node_id are required");
                    return;
                  }
                  void api
                    .rotateNodeCertificate({ tenant_id: tenantId.trim(), node_id: nodeId.trim() })
                    .then(load)
                    .catch((err) => setError(err instanceof Error ? err.message : "rotate failed"));
                }}
              >
                Rotate
              </ActionButton>
              <ActionButton
                variant="secondary"
                onClick={() => {
                  if (!tenantId.trim() || !nodeId.trim()) {
                    setError("tenant_id and node_id are required");
                    return;
                  }
                  void api
                    .renewNodeCertificate({ tenant_id: tenantId.trim(), node_id: nodeId.trim() })
                    .then(load)
                    .catch((err) => setError(err instanceof Error ? err.message : "renew failed"));
                }}
              >
                Renew
              </ActionButton>
              <ActionButton
                variant="danger"
                onClick={() => {
                  if (!tenantId.trim() || !nodeId.trim()) {
                    setError("tenant_id and node_id are required");
                    return;
                  }
                  void api
                    .revokeNodeCertificate({ tenant_id: tenantId.trim(), node_id: nodeId.trim() })
                    .then(load)
                    .catch((err) => setError(err instanceof Error ? err.message : "revoke failed"));
                }}
              >
                Revoke
              </ActionButton>
              <SupportBadge state="frontend_seam" />
            </div>
          </Card>
        ) : null}
      </AdminShell>
    </RouteGuard>
  );
}