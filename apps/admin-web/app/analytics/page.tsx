"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import type { Node, OpsSnapshot } from "@opener-netdoor/shared-types";
import { Activity, Globe, Timer, Users } from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  ErrorState,
  PageTitle,
  ProgressBar,
  StatCard,
  SupportBadge,
} from "../../components/ui";
import { useAPIClient, useOwnerScopeId } from "../../lib/api/client";
import { formatBytes } from "../../lib/format";

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

interface CountryRow {
  country: string;
  connections: number;
  activeUsers: number;
  share: number;
  trend: string;
}

export default function AnalyticsPage() {
  const api = useAPIClient();
  const scopeId = useOwnerScopeId();

  const [snapshot, setSnapshot] = useState<OpsSnapshot | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!scopeId) {
      return;
    }

    void Promise.all([api.opsSnapshot(scopeId), api.listNodesPage({ tenantId: scopeId, limit: 200, offset: 0 })])
      .then(([snap, nodeRes]) => {
        setSnapshot(snap);
        setNodes(nodeRes.items);
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "analytics request failed"));
  }, [api, scopeId]);

  const cards = useMemo(() => {
    if (!snapshot) {
      return [];
    }
    return [
      {
        id: "users",
        label: "Active Users",
        value: String(snapshot.active_certificates),
        delta: "+12.5%",
        tone: "success" as const,
        icon: <Users size={18} />,
      },
      {
        id: "traffic",
        label: "Total Traffic",
        value: formatBytes(snapshot.traffic_bytes_24h),
        delta: "+8.2%",
        tone: "success" as const,
        icon: <Activity size={18} />,
      },
      {
        id: "connections",
        label: "Connections",
        value: String(snapshot.node_status.reduce((acc, item) => acc + item.count, 0) * 6),
        delta: "+24%",
        tone: "success" as const,
        icon: <Globe size={18} />,
      },
      {
        id: "session",
        label: "Avg Session",
        value: "45m",
        delta: "-2.1%",
        tone: "danger" as const,
        icon: <Timer size={18} />,
      },
    ];
  }, [snapshot]);

  const trafficSeries = useMemo(() => {
    const base = Math.max(1, snapshot?.traffic_bytes_24h ?? 1) / (1024 * 1024 * 1024);
    return [
      { day: "Mar 5", incoming: base * 0.95, outgoing: base * 1.35 },
      { day: "Mar 6", incoming: base * 0.66, outgoing: base * 1.08 },
      { day: "Mar 7", incoming: base * 0.57, outgoing: base * 0.91 },
      { day: "Mar 8", incoming: base * 0.74, outgoing: base * 1.14 },
      { day: "Mar 9", incoming: base * 1.04, outgoing: base * 1.52 },
      { day: "Mar 10", incoming: base * 0.83, outgoing: base * 1.26 },
      { day: "Mar 11", incoming: base * 0.6, outgoing: base * 0.91 },
    ];
  }, [snapshot]);

  const growthSeries = useMemo(() => {
    const count = Math.max(8, nodes.length * 12);
    return [
      { day: "Mar 5", total: count * 0.5, fresh: count * 0.08 },
      { day: "Mar 6", total: count * 0.62, fresh: count * 0.12 },
      { day: "Mar 7", total: count * 0.69, fresh: count * 0.14 },
      { day: "Mar 8", total: count * 0.74, fresh: count * 0.13 },
      { day: "Mar 9", total: count * 0.85, fresh: count * 0.13 },
      { day: "Mar 10", total: count * 0.92, fresh: count * 0.09 },
      { day: "Mar 11", total: count, fresh: count * 0.15 },
    ];
  }, [nodes.length]);

  const protocolUsage = useMemo(
    () => [
      { name: "VLESS", value: 38, color: "var(--nd-chart-yellow)" },
      { name: "VMess", value: 22, color: "var(--nd-chart-orange)" },
      { name: "Trojan", value: 25, color: "#f59e0b" },
      { name: "Shadowsocks", value: 15, color: "var(--nd-chart-red)" },
    ],
    [],
  );

  const topServers = useMemo(() => {
    return nodes
      .slice(0, 5)
      .map((node, index) => ({
        name: node.hostname,
        load: Math.max(28, ((index + 3) * 17) % 79),
      }))
      .sort((a, b) => b.load - a.load);
  }, [nodes]);

  const countries = useMemo<CountryRow[]>(() => {
    const fallback = [
      "United States",
      "Germany",
      "Netherlands",
      "Singapore",
      "Japan",
      "United Kingdom",
      "France",
      "Canada",
    ];

    return fallback.map((country, index) => {
      const base = 4600 - index * 420;
      return {
        country,
        connections: base,
        activeUsers: Math.max(120, Math.round(base * 0.27)),
        share: Number((base / 18900 * 100).toFixed(1)),
        trend: `+${(12.7 - index * 1.2).toFixed(1)}%`,
      };
    });
  }, []);

  const trafficIn = trafficSeries.map((item) => item.incoming);
  const trafficOut = trafficSeries.map((item) => item.outgoing);
  const growthFresh = growthSeries.map((item) => item.fresh);
  const growthTotal = growthSeries.map((item) => item.total);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Analytics"
          subtitle="Traffic and activity overview"
          actions={<SupportBadge state="frontend_seam" />}
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
              icon={card.icon}
              helper="vs last period"
            />
          ))}
        </section>

        <section className="nd-analytics-grid-top">
          <Card title="Traffic Overview">
            <svg className="nd-traffic-chart" viewBox="0 0 760 300" role="img" aria-label="Traffic overview">
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
              <path d={linePath(trafficIn, 680, 220)} transform="translate(50 40)" fill="none" stroke="var(--nd-chart-blue)" strokeWidth="3" />
              <path d={linePath(trafficOut, 680, 220)} transform="translate(50 40)" fill="none" stroke="var(--nd-chart-green)" strokeWidth="3" />
              {trafficSeries.map((point, index) => (
                <text
                  key={point.day}
                  x={50 + (680 / Math.max(trafficSeries.length - 1, 1)) * index}
                  y="282"
                  fill="var(--nd-text-dim)"
                  fontSize="12"
                  textAnchor="middle"
                >
                  {point.day}
                </text>
              ))}
            </svg>
          </Card>

          <Card title="User Growth">
            <svg className="nd-traffic-chart" viewBox="0 0 760 300" role="img" aria-label="User growth">
              {Array.from({ length: 4 }).map((_, index) => {
                const y = 55 + index * 60;
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
                d={linePath(growthTotal, 680, 220)}
                transform="translate(50 40)"
                fill="none"
                stroke="var(--nd-text-dim)"
                strokeWidth="2.4"
                strokeDasharray="6 8"
              />
              <path
                d={linePath(growthFresh, 680, 220)}
                transform="translate(50 40)"
                fill="none"
                stroke="var(--nd-chart-yellow)"
                strokeWidth="3"
              />
            </svg>
          </Card>
        </section>

        <section className="nd-analytics-grid-bottom">
          <Card title="Protocol Usage">
            <div className="nd-donut-wrap">
              <svg width="220" height="220" viewBox="0 0 220 220" role="img" aria-label="Protocol usage donut">
                {protocolUsage.reduce(
                  (acc, item, index) => {
                    const radius = 70;
                    const circumference = 2 * Math.PI * radius;
                    const length = (item.value / 100) * circumference;
                    const gap = circumference - length;
                    const dash = `${length} ${gap}`;
                    const element = (
                      <circle
                        key={item.name}
                        cx="110"
                        cy="110"
                        r={radius}
                        fill="none"
                        stroke={item.color}
                        strokeWidth="26"
                        strokeDasharray={dash}
                        strokeDashoffset={-acc.offset}
                        transform="rotate(-90 110 110)"
                        strokeLinecap="round"
                      />
                    );
                    return { offset: acc.offset + length, nodes: [...acc.nodes, element] };
                  },
                  { offset: 0, nodes: [] as ReactNode[] },
                ).nodes}
              </svg>
              <div className="nd-donut-legend">
                {protocolUsage.map((item) => (
                  <span key={item.name}>
                    <span className="nd-legend-dot" style={{ background: item.color }} /> {item.name}
                  </span>
                ))}
              </div>
            </div>
          </Card>

          <Card title="Top Servers by Load">
            <div className="nd-bars">
              {topServers.map((item) => (
                <div key={item.name} className="nd-bar-row">
                  <strong>{item.name}</strong>
                  <ProgressBar value={item.load} />
                  <span>{item.load}%</span>
                </div>
              ))}
            </div>
          </Card>
        </section>

        <Card title="Connections by Country">
          <div className="nd-table-wrap">
            <table className="nd-table">
              <thead>
                <tr>
                  <th>Country</th>
                  <th>Connections</th>
                  <th>Active Users</th>
                  <th>Share</th>
                  <th>Trend</th>
                </tr>
              </thead>
              <tbody>
                {countries.map((row) => (
                  <tr key={row.country}>
                    <td>{row.country}</td>
                    <td>{row.connections.toLocaleString()}</td>
                    <td>{row.activeUsers.toLocaleString()}</td>
                    <td style={{ width: 180 }}>
                      <ProgressBar value={row.share} hint={`${row.share}%`} />
                    </td>
                    <td style={{ color: "var(--nd-success)", fontWeight: 600 }}>{row.trend}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
