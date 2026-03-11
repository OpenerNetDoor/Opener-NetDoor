"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type {
  AuditLogRecord,
  HealthResponse,
  Node,
  OpsAnalytics,
  OpsSnapshot,
  User,
} from "@opener-netdoor/shared-types";
import { ArrowUpRight, Plus, RefreshCw } from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  EmptyState,
  ErrorState,
  LoadingState,
  PageTitle,
  StatCard,
  StatusBadge,
} from "../../components/ui";
import { useAPIClient, useOwnerScopeId } from "../../lib/api/client";
import { buildDashboardCards, DASHBOARD_ACTIONS } from "../../lib/adapters/dashboard";
import { formatBytes, formatRelativeTime } from "../../lib/format";
import { SupportActionList } from "../../components/domain";
import { subscribeAdminDataChanged } from "../../lib/events/admin-data";

interface ActivityRow {
  id: string;
  user: string;
  action: string;
  server: string;
  when: string;
  traffic: string;
}

function linePath(values: number[], width: number, height: number): string {
  const max = Math.max(...values, 1);
  const stepX = width / Math.max(values.length - 1, 1);
  return values
    .map((value, index) => {
      const x = index * stepX;
      const y = height - (value / max) * height;
      return `${index === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
}

function metadataString(record: AuditLogRecord, key: string): string | undefined {
  const value = record.metadata?.[key];
  if (typeof value === "string" && value.trim().length > 0) {
    return value;
  }
  return undefined;
}

function metadataNumber(record: AuditLogRecord, key: string): number | undefined {
  const value = record.metadata?.[key];
  if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
    return value;
  }
  return undefined;
}

export default function DashboardPage() {
  const api = useAPIClient();
  const scopeId = useOwnerScopeId();

  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [ready, setReady] = useState<HealthResponse | null>(null);
  const [snapshot, setSnapshot] = useState<OpsSnapshot | null>(null);
  const [analytics, setAnalytics] = useState<OpsAnalytics | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [recentAudit, setRecentAudit] = useState<AuditLogRecord[]>([]);
  const [error, setError] = useState("");

  const loadDashboard = useCallback(async () => {
    if (!scopeId) {
      return;
    }
    try {
      const [h, r, s, a] = await Promise.all([
        api.health(),
        api.ready(),
        api.opsSnapshot(scopeId),
        api.opsAnalytics(scopeId),
      ]);

      const [usersRes, nodesRes, auditRes] = await Promise.allSettled([
        api.listUsersPage({ tenantId: scopeId, limit: 100, offset: 0 }),
        api.listNodesPage({ tenantId: scopeId, limit: 100, offset: 0 }),
        api.listAuditLogs({ tenantId: scopeId, limit: 12, offset: 0 }),
      ]);

      setHealth(h);
      setReady(r);
      setSnapshot(s);
      setAnalytics(a);
      setUsers(usersRes.status === "fulfilled" ? usersRes.value.items : []);
      setNodes(nodesRes.status === "fulfilled" ? nodesRes.value.items : []);
      setRecentAudit(auditRes.status === "fulfilled" ? auditRes.value.items : []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "dashboard request failed");
    }
  }, [api, scopeId]);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      if (cancelled) {
        return;
      }
      await loadDashboard();
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [loadDashboard]);

  useEffect(() => {
    return subscribeAdminDataChanged(() => {
      void loadDashboard();
    });
  }, [loadDashboard]);

  const cards = useMemo(
    () =>
      buildDashboardCards({
        fallbackUserCount: users.filter((item) => item.status === "active").length,
        fallbackNodeCount: nodes.length,
        analytics,
      }),
    [analytics, nodes.length, users],
  );

  const traffic = useMemo(() => analytics?.traffic_history_7d ?? [], [analytics]);
  const incoming = useMemo(() => traffic.map((item) => item.bytes_in / (1024 * 1024 * 1024)), [traffic]);
  const outgoing = useMemo(() => traffic.map((item) => item.bytes_out / (1024 * 1024 * 1024)), [traffic]);

  const serverStatus = useMemo(
    () =>
      nodes.slice(0, 4).map((node) => ({
        id: node.id,
        code: node.region.slice(0, 2).toUpperCase() || "--",
        name: node.hostname,
        detail: node.last_heartbeat_at ? `heartbeat ${formatRelativeTime(node.last_heartbeat_at)}` : "no heartbeat yet",
        status: node.status,
      })),
    [nodes],
  );

  const activity = useMemo<ActivityRow[]>(() => {
    return recentAudit.slice(0, 6).map((item) => ({
      id: item.id,
      user: metadataString(item, "email") ?? item.actor_type ?? "No data",
      action: item.action,
      server: metadataString(item, "hostname") ?? item.target_id ?? "No data",
      when: formatRelativeTime(item.created_at),
      traffic: (() => {
        const bytes = metadataNumber(item, "bytes_total");
        return typeof bytes === "number" ? formatBytes(bytes) : "No data";
      })(),
    }));
  }, [recentAudit]);

  const maxY = Math.max(...incoming, ...outgoing, 1);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Dashboard"
          subtitle="Overview of your VPN service performance"
          actions={<span style={{ color: "var(--nd-text-muted)", fontSize: 12 }}>Last updated: {formatRelativeTime(analytics?.generated_at)}</span>}
        />

        {error ? <ErrorState message={error} /> : null}

        <section className="nd-stat-grid">
          {cards.map((card) => (
            <StatCard
              key={card.id}
              label={card.label}
              value={card.value}
              delta={card.delta}
              tone={card.tone}
              icon={<card.icon size={18} strokeWidth={2} />}
            />
          ))}
        </section>

        <section className="nd-dashboard-grid">
          <Card
            title="Traffic Overview"
            subtitle="Network traffic for the last 7 days"
            actions={
              <div className="nd-chart-legend">
                <span>
                  <span className="nd-legend-dot" style={{ background: "var(--nd-chart-blue)" }} /> Incoming
                </span>
                <span>
                  <span className="nd-legend-dot" style={{ background: "var(--nd-chart-green)" }} /> Outgoing
                </span>
              </div>
            }
          >
            {analytics === null ? (
              <LoadingState label="Loading traffic metrics..." />
            ) : traffic.length === 0 ? (
              <EmptyState title="No data" description="Traffic history is not available yet." />
            ) : (
              <svg className="nd-traffic-chart" viewBox="0 0 760 300" role="img" aria-label="Traffic chart">
                {Array.from({ length: 5 }).map((_, index) => {
                  const y = 40 + index * 55;
                  return (
                    <line
                      key={y}
                      x1="50"
                      x2="730"
                      y1={y}
                      y2={y}
                      stroke="var(--nd-chart-grid)"
                      strokeWidth="1"
                      strokeDasharray="4 6"
                    />
                  );
                })}

                <path
                  d={linePath(outgoing, 680, 220)}
                  transform="translate(50 40)"
                  fill="none"
                  stroke="var(--nd-chart-green)"
                  strokeWidth="3"
                  strokeLinecap="round"
                />
                <path
                  d={linePath(incoming, 680, 220)}
                  transform="translate(50 40)"
                  fill="none"
                  stroke="var(--nd-chart-blue)"
                  strokeWidth="3"
                  strokeLinecap="round"
                />

                {traffic.map((point, index) => (
                  <text
                    key={point.ts_hour}
                    x={50 + (680 / Math.max(traffic.length - 1, 1)) * index}
                    y="282"
                    fill="var(--nd-text-dim)"
                    fontSize="12"
                    textAnchor="middle"
                  >
                    {new Date(point.ts_hour).toLocaleDateString([], { month: "short", day: "numeric" })}
                  </text>
                ))}
                <text x="44" y="40" fill="var(--nd-text-dim)" fontSize="11">
                  {Math.round(maxY * 1.2)} GB
                </text>
              </svg>
            )}
          </Card>

          <div style={{ display: "grid", gap: 20 }}>
            <Card title="Quick Actions">
              <SupportActionList items={DASHBOARD_ACTIONS} />
            </Card>

            <Card title="Server Status">
              {serverStatus.length === 0 ? (
                <EmptyState title="No data" description="No servers available yet." />
              ) : (
                <div className="nd-server-list">
                  {serverStatus.map((item) => (
                    <article key={item.id} className="nd-server-item">
                      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                        <span className="nd-server-flag">{item.code}</span>
                        <div>
                          <strong>{item.name}</strong>
                          <div style={{ color: "var(--nd-text-muted)", fontSize: 12 }}>{item.detail}</div>
                        </div>
                      </div>
                      <StatusBadge value={item.status === "active" ? "Online" : item.status} />
                    </article>
                  ))}
                </div>
              )}
            </Card>
          </div>
        </section>

        <Card
          title="Recent Activity"
          actions={
            <Link href="/analytics" style={{ color: "var(--nd-accent-blue)", fontSize: 13 }}>
              View all
            </Link>
          }
          className="nd-activity-table"
        >
          {activity.length === 0 ? (
            <EmptyState title="No data" description="No recent activity events available." />
          ) : (
            <div className="nd-table-wrap">
              <table className="nd-table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Action</th>
                    <th>Server</th>
                    <th>Time</th>
                    <th>Traffic</th>
                  </tr>
                </thead>
                <tbody>
                  {activity.map((row) => (
                    <tr key={row.id}>
                      <td>
                        <div className="nd-user-cell">
                          <span className="nd-avatar">{row.user[0]?.toUpperCase() ?? "U"}</span>
                          <span>{row.user}</span>
                        </div>
                      </td>
                      <td>
                        <StatusBadge value={row.action} />
                      </td>
                      <td>{row.server}</td>
                      <td>{row.when}</td>
                      <td>{row.traffic}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div style={{ marginTop: 14, display: "flex", gap: 8, flexWrap: "wrap" }}>
            <button className="nd-btn is-secondary" type="button" disabled>
              <Plus size={15} /> Add Server (frontend_seam)
            </button>
            <button className="nd-btn is-secondary" type="button" disabled>
              <RefreshCw size={15} /> Restart Server (frontend_seam)
            </button>
            <Link href="/analytics">
              <button className="nd-btn is-ghost" type="button">
                <ArrowUpRight size={15} /> Open Analytics
              </button>
            </Link>
          </div>

          <div style={{ marginTop: 10, display: "flex", gap: 10, flexWrap: "wrap" }}>
            <StatusBadge value={`health ${health?.status ?? "unknown"}`} />
            <StatusBadge value={`ready ${ready?.status ?? "unknown"}`} />
            <StatusBadge value={`snapshot ${snapshot ? "ok" : "n/a"}`} />
          </div>
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
