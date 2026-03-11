"use client";

import { useEffect, useMemo, useState } from "react";
import type { AccessKey } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  ConfirmDangerButton,
  DataTable,
  ErrorState,
  PageTitle,
  PaginationControls,
  SearchInput,
  StatusBadge,
  SupportBadge,
} from "../../components/ui";
import { useAPIClient, useAdminSession } from "../../lib/api/client";
import { hasScopes } from "../../lib/permissions";

export default function AccessKeysPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<AccessKey[]>([]);
  const [error, setError] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  const load = () => {
    void api
      .listAccessKeysPage({
        status: status || undefined,
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

  const filtered = useMemo(() => {
    if (!search.trim()) {
      return items;
    }
    const query = search.trim().toLowerCase();
    return items.filter((item) => item.id.toLowerCase().includes(query) || item.user_id.toLowerCase().includes(query));
  }, [items, search]);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Keys"
          subtitle="Access keys for your users. Create and revoke flows are available in Users -> Access Keys modal."
          actions={<SupportBadge state="supported" />}
        />

        <Card title="Filters">
          <div className="row">
            <select value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">all statuses</option>
              <option value="active">active</option>
              <option value="revoked">revoked</option>
            </select>
            <SearchInput value={search} onChange={setSearch} placeholder="Search by key or user id" />
            <button className="nd-btn is-primary" onClick={load} type="button">
              Apply
            </button>
          </div>
        </Card>

        <Card title="Keys list">
          {error ? <ErrorState message={error} /> : null}
          <DataTable
            rows={filtered}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "Key", render: (row) => <code>{row.id}</code> },
              { id: "user", header: "User", render: (row) => <code>{row.user_id}</code> },
              { id: "type", header: "Type", render: (row) => row.key_type },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "created", header: "Created", render: (row) => row.created_at },
              {
                id: "actions",
                header: "Actions",
                render: (row) =>
                  canWrite && row.status !== "revoked" ? (
                    <ConfirmDangerButton
                      label="Revoke"
                      prompt={`Revoke key ${row.id}?`}
                      onConfirm={async () => {
                        await api.revokeAccessKey(row.id, row.tenant_id);
                        load();
                      }}
                    />
                  ) : (
                    "read-only"
                  ),
              },
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

