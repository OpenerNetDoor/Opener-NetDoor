"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { Tenant } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { Card, DataTable, ErrorState, PageTitle, PaginationControls, StatusBadge } from "../../components/ui";
import { useAPIClient } from "../../lib/api/client";

export default function TenantsPage() {
  const api = useAPIClient();
  const [items, setItems] = useState<Tenant[]>([]);
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    void api
      .listTenantsPage({ limit, offset, status: status || undefined })
      .then((result) => {
        if (cancelled) {
          return;
        }
        setItems(result.items);
        setError("");
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "request failed");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [api, limit, offset, status]);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle title="Tenants" subtitle="Tenant inventory with scope-aware navigation." />
        <Card
          title="Filters"
          actions={
            <div className="row" style={{ width: 280 }}>
              <select value={status} onChange={(event) => setStatus(event.target.value)}>
                <option value="">all statuses</option>
                <option value="active">active</option>
                <option value="suspended">suspended</option>
              </select>
            </div>
          }
        >
          {error ? <ErrorState message={error} /> : null}
          <DataTable
            rows={items}
            rowKey={(row) => row.id}
            columns={[
              {
                id: "name",
                header: "Tenant",
                render: (row) => <Link href={`/tenants/${row.id}`}>{row.name}</Link>,
              },
              { id: "id", header: "Tenant ID", render: (row) => <code>{row.id}</code> },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "created", header: "Created", render: (row) => row.created_at },
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
