"use client";

import { FormEvent, useEffect, useState } from "react";
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
import { toIssuersVM, type IssuerVM } from "../../../lib/adapters/pki";

export default function PKIIssuersPage() {
  const api = useAPIClient();
  const session = useAdminSession();
  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<IssuerVM[]>([]);
  const [error, setError] = useState("");

  const [issuerId, setIssuerId] = useState("");
  const [source, setSource] = useState<"file" | "external">("file");
  const [activateNow, setActivateNow] = useState(true);
  const [activateIssuerId, setActivateIssuerId] = useState("");

  const load = () => {
    void api
      .listPKIIssuersPage({ limit, offset })
      .then((result) => {
        setItems(toIssuersVM(result.items));
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  useEffect(() => {
    load();
  }, [limit, offset]);

  const createIssuer = (event: FormEvent) => {
    event.preventDefault();
    if (!issuerId.trim()) {
      setError("issuer_id is required");
      return;
    }
    void api
      .createPKIIssuer({
        issuer_id: issuerId.trim(),
        source,
        activate: activateNow,
      })
      .then(() => {
        setIssuerId("");
        setError("");
        load();
      })
      .catch((err) => setError(err instanceof Error ? err.message : "create issuer failed"));
  };

  return (
    <RouteGuard requiredScopes={["admin:read"]} platformOnly>
      <AdminShell>
        <PageTitle
          title="PKI Issuers"
          subtitle="Issuer lifecycle with overlap-safe activation and auditability."
          actions={<SupportBadge state="supported" />}
        />

        {error ? <ErrorState message={error} /> : null}

        <Card title="Issuer inventory" actions={<button className="nd-btn is-secondary" onClick={load}>Refresh</button>}>
          <DataTable
            rows={items}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "ID", render: (row) => <code>{row.id}</code> },
              { id: "issuer", header: "Issuer", render: (row) => row.issuerId },
              { id: "source", header: "Source", render: (row) => row.source },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "activated", header: "Activated", render: (row) => row.activatedAt },
              { id: "retired", header: "Retired", render: (row) => row.retiredAt },
              { id: "created", header: "Created", render: (row) => row.createdAt },
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
          <div className="grid-two">
            <Card title="Create issuer" subtitle="Supports file and external metadata issuers.">
              <form onSubmit={createIssuer} className="row">
                <input placeholder="issuer_id" value={issuerId} onChange={(event) => setIssuerId(event.target.value)} />
                <select value={source} onChange={(event) => setSource(event.target.value as "file" | "external")}> 
                  <option value="file">file</option>
                  <option value="external">external</option>
                </select>
                <label>
                  <input type="checkbox" checked={activateNow} onChange={(event) => setActivateNow(event.target.checked)} /> activate now
                </label>
                <ActionButton type="submit">Create issuer</ActionButton>
              </form>
            </Card>

            <Card title="Activate issuer" subtitle="Switch active issuer with overlap trust from previous set.">
              <div className="row">
                <input placeholder="issuer_id" value={activateIssuerId} onChange={(event) => setActivateIssuerId(event.target.value)} />
                <ActionButton
                  onClick={() => {
                    if (!activateIssuerId.trim()) {
                      setError("issuer_id is required");
                      return;
                    }
                    void api
                      .activatePKIIssuer({ issuer_id: activateIssuerId.trim() })
                      .then(() => {
                        setError("");
                        load();
                      })
                      .catch((err) => setError(err instanceof Error ? err.message : "activation failed"));
                  }}
                >
                  Activate
                </ActionButton>
              </div>
            </Card>
          </div>
        ) : null}
      </AdminShell>
    </RouteGuard>
  );
}