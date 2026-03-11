"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  DataTable,
  ErrorState,
  LogViewer,
  NotificationItem,
  PageTitle,
  PaginationControls,
  SearchInput,
  SupportBadge,
} from "../../components/ui";
import { useAPIClient, useAdminSession } from "../../lib/api/client";
import { toAuditEventVMs } from "../../lib/adapters/audit";
import { buildHeaderNotifications, type NotificationView } from "../../lib/mock/notifications";

export default function AuditLogsPage() {
  const api = useAPIClient();
  const session = useAdminSession();

  const [tenantId, setTenantId] = useState("");
  const [action, setAction] = useState("");
  const [actorType, setActorType] = useState("");
  const [targetType, setTargetType] = useState("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<ReturnType<typeof toAuditEventVMs>>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  const load = () => {
    void api
      .listAuditLogs({
        tenantId: tenantId || undefined,
        action: action || undefined,
        actorType: actorType || undefined,
        targetType: targetType || undefined,
        limit,
        offset,
      })
      .then((response) => {
        setItems(toAuditEventVMs(response.items));
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "request failed"));
  };

  useEffect(() => {
    load();
  }, [limit, offset]);

  const feed: NotificationView[] = useMemo(() => {
    if (items.length > 0) {
      return items.slice(0, 8).map((item) => ({
        id: item.id,
        title: item.action,
        subtitle: `${item.actor} -> ${item.target}`,
        when: item.relative,
        tone: item.action.includes("revoke") || item.action.includes("reject") ? ("danger" as const) : ("neutral" as const),
      }));
    }
    return buildHeaderNotifications("/audit-logs");
  }, [items]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle
          title="Audit Logs"
          subtitle="Timeline + table view for actor and node mutations."
          actions={<SupportBadge state="supported" />}
        />

        <Card title="Filters">
          <div className="row">
            <input placeholder="tenant_id" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <SearchInput value={action} onChange={setAction} placeholder="action" />
            <SearchInput value={actorType} onChange={setActorType} placeholder="actor_type" />
            <SearchInput value={targetType} onChange={setTargetType} placeholder="target_type" />
            <button className="nd-btn is-primary" onClick={load} type="button">
              Apply
            </button>
          </div>
        </Card>

        {error ? <ErrorState message={error} /> : null}

        <div className="grid-two">
          <Card title="Event timeline" subtitle="Notification-style compact feed.">
            {feed.map((event) => (
              <NotificationItem key={event.id} item={event} />
            ))}
          </Card>

          <Card title="Raw payload preview" subtitle="Expanded metadata for incident response.">
            <LogViewer logs={items} />
          </Card>
        </div>

        <Card title="Audit records">
          <DataTable
            rows={items}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "ID", render: (row) => <code>{row.id}</code> },
              { id: "tenant", header: "Tenant", render: (row) => row.tenant },
              { id: "action", header: "Action", render: (row) => row.action },
              { id: "actor", header: "Actor", render: (row) => row.actor },
              { id: "target", header: "Target", render: (row) => row.target },
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
      </AdminShell>
    </RouteGuard>
  );
}