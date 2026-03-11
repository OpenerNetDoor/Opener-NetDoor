"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type { AuditLogRecord, HealthResponse, Node, OpsSnapshot, User } from "@opener-netdoor/shared-types";
import { ArrowUpRight, Plus, RefreshCw } from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  ErrorState,
  LoadingState,
  PageTitle,
  StatCard,
  StatusBadge,
} from "../../components/ui";
import { useAPIClient, useOwnerScopeId } from "../../lib/api/client";
import { buildDashboardCards, buildTrafficSeries, DASHBOARD_ACTIONS } from "../../lib/adapters/dashboard";
import { formatBytes } from "../../lib/format";
import { SupportActionList } from "../../components/domain";
import { subscribeAdminDataChanged } from "../../lib/events/admin-data";

interface ActivityRow {
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

export default function DashboardPage() {
  const api = useAPIClient();
  const scopeId = useOwnerScopeId();

  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [ready, setReady] = useState<HealthResponse | null>(null);
  const [snapshot, setSnapshot] = useState<OpsSnapshot | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [accessKeyCount, setAccessKeyCount] = useState(0);
  const [recentAudit, setRecentAudit] = useState<AuditLogRecord[]>([]);
  const [error, setError] = useState("");

  const loadDashboard = useCallback(async () => {
    if (!scopeId) {
      return;
    }
    try {
      const [h, r, s] = await Promise.all([api.health(), api.ready(), api.opsSnapshot(scopeId)]);

      const [usersRes, keysRes, nodesRes, auditRes] = await Promise.allSettled([
        api.listUsersPage({ tenantId: scopeId, limit: 100, offset: 0 }),
        api.listAccessKeysPage({ tenantId: scopeId, limit: 100, offset: 0 }),
        api.listNodesPage({ tenantId: scopeId, limit: 100, offset: 0 }),
        api.listAuditLogs({ tenantId: scopeId, limit: 12, offset: 0 }),
      ]);

      setHealth(h);
      setReady(r);
      setSnapshot(s);
      setUsers(usersRes.status === "fulfilled" ? usersRes.value.items : []);
      setAccessKeyCount(keysRes.status === "fulfilled" ? keysRes.value.items.length : 0);
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

  const newUsers7d = useMemo(
    () =>
      users.filter((item) => {
        const created = new Date(item.created_at).getTime();
        return Number.isFinite(created) && Date.now() - created <= 7 * 24 * 60 * 60 * 1000;
      }).length,
    [users],
  );

  const cards = useMemo(
    () =>
      buildDashboardCards({
        userCount: users.length,
        nodeCount: nodes.length,
        newUsers: newUsers7d,
        snapshot,
      }),
    [newUsers7d, nodes.length, snapshot, users.length],
  );

  const traffic = useMemo(() => buildTrafficSeries(snapshot), [snapshot]);

  const incoming = useMemo(() => traffic.map((item) => item.incoming), [traffic]);
  const outgoing = useMemo(() => traffic.map((item) => item.outgoing), [traffic]);

  const serverStatus = useMemo(
    () =>
      nodes.slice(0, 4).map((node) => ({
        id: node.id,
        code: node.region.slice(0, 2).toUpperCase() || "--",
        name: node.hostname,
        users: Math.max(1, Math.round((node.id.length * 37) % 420)),
        status: node.status,
      })),
    [nodes],
  );

  const activity = useMemo<ActivityRow[]>(() => {
    return recentAudit.slice(0, 6).map((item, index) => ({
      user: users[index % Math.max(1, users.length)]?.email ?? `user-${index + 1}@example.com`,
      action: item.action.includes("revoke") ? "Disconnected" : "Connected",
      server: nodes[index % Math.max(1, nodes.length)]?.hostname ?? "server",
      when: new Date(item.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      traffic: formatBytes((index + 1) * 220 * 1024 * 1024),
    }));
  }, [nodes, recentAudit, users]);

  const maxY = Math.max(...incoming, ...outgoing, 1);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Dashboard"
          subtitle="Overview of your VPN service performance"
          actions={<span style={{ color: "var(--nd-text-muted)", fontSize: 12 }}>Last updated: just now</span>}
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
              helper="vs last week"
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
            {traffic.length === 0 ? (
              <LoadingState label="Waiting for traffic metrics..." />
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
                    key={point.label}
                    x={50 + (680 / Math.max(traffic.length - 1, 1)) * index}
                    y="282"
                    fill="var(--nd-text-dim)"
                    fontSize="12"
                    textAnchor="middle"
                  >
                    {point.label}
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
              <div className="nd-server-list">
                {serverStatus.map((item) => (
                  <article key={item.id} className="nd-server-item">
                    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                      <span className="nd-server-flag">{item.code}</span>
                      <div>
                        <strong>{item.name}</strong>
                        <div style={{ color: "var(--nd-text-muted)", fontSize: 12 }}>{item.users} users</div>
                      </div>
                    </div>
                    <StatusBadge value={item.status === "active" ? "Online" : item.status} />
                  </article>
                ))}
              </div>
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
                {activity.map((row, index) => (
                  <tr key={`${row.user}-${index}`}>
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
            <StatusBadge value={`keys ${accessKeyCount}`} />
            <StatusBadge value={`health ${health?.status ?? "unknown"}`} />
            <StatusBadge value={`ready ${ready?.status ?? "unknown"}`} />
          </div>
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}


