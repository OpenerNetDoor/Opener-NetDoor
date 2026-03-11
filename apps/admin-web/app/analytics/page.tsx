"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import type { OpsAnalytics } from "@opener-netdoor/shared-types";
import { Activity, Globe, Timer, Users } from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  Card,
  EmptyState,
  ErrorState,
  LoadingState,
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

export default function AnalyticsPage() {
  const api = useAPIClient();
  const scopeId = useOwnerScopeId();

  const [analytics, setAnalytics] = useState<OpsAnalytics | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!scopeId) {
      return;
    }

    void api
      .opsAnalytics(scopeId)
      .then((data) => {
        setAnalytics(data);
        setError("");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "analytics request failed"));
  }, [api, scopeId]);

  const cards = useMemo(() => {
    if (!analytics) {
      return [];
    }
    return [
      {
        id: "users",
        label: "Active Users",
        value: String(analytics.active_users),
        tone: "success" as const,
        icon: <Users size={18} />,
      },
      {
        id: "traffic",
        label: "Traffic (24h)",
        value: formatBytes(analytics.traffic_bytes_24h),
        tone: "success" as const,
        icon: <Activity size={18} />,
      },
      {
        id: "keys",
        label: "Active Keys",
        value: String(analytics.active_keys),
        tone: "neutral" as const,
        icon: <Globe size={18} />,
      },
      {
        id: "servers",
        label: "Servers Online",
        value: String(analytics.online_servers),
        tone: "neutral" as const,
        icon: <Timer size={18} />,
      },
    ];
  }, [analytics]);

  const trafficSeries = analytics?.traffic_history_7d ?? [];
  const trafficIn = trafficSeries.map((item) => item.bytes_in / (1024 * 1024 * 1024));
  const trafficOut = trafficSeries.map((item) => item.bytes_out / (1024 * 1024 * 1024));

  const growthSeries = analytics?.user_growth_7d ?? [];
  const growthFresh = growthSeries.map((item) => item.new_users);
  const growthTotal = growthSeries.map((item) => item.total_users);

  const protocolUsage = analytics?.protocol_usage_24h ?? [];
  const protocolColors = ["var(--nd-chart-yellow)", "var(--nd-chart-orange)", "#f59e0b", "var(--nd-chart-red)", "var(--nd-chart-blue)"];

  const topServers = analytics?.top_servers_by_load ?? [];

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Analytics"
          subtitle="Traffic and activity overview"
          actions={<SupportBadge state="supported" />}
        />

        {error ? <ErrorState message={error} /> : null}
        {!analytics && !error ? <LoadingState label="Loading analytics..." /> : null}

        <section className="nd-stat-grid">
          {cards.map((card) => (
            <StatCard
              key={card.id}
              label={card.label}
              value={card.value}
              tone={card.tone}
              icon={card.icon}
            />
          ))}
        </section>

        <section className="nd-analytics-grid-top">
          <Card title="Traffic Overview">
            {trafficSeries.length === 0 ? (
              <EmptyState title="No data" description="No traffic history for the selected period." />
            ) : (
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
                    key={point.ts_hour}
                    x={50 + (680 / Math.max(trafficSeries.length - 1, 1)) * index}
                    y="282"
                    fill="var(--nd-text-dim)"
                    fontSize="12"
                    textAnchor="middle"
                  >
                    {new Date(point.ts_hour).toLocaleDateString([], { month: "short", day: "numeric" })}
                  </text>
                ))}
              </svg>
            )}
          </Card>

          <Card title="User Growth">
            {growthSeries.length === 0 ? (
              <EmptyState title="No data" description="No user growth history for the selected period." />
            ) : (
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
            )}
          </Card>
        </section>

        <section className="nd-analytics-grid-bottom">
          <Card title="Protocol Usage">
            {protocolUsage.length === 0 ? (
              <EmptyState title="No data" description="Protocol usage is empty for the last 24h." />
            ) : (
              <div className="nd-donut-wrap">
                <svg width="220" height="220" viewBox="0 0 220 220" role="img" aria-label="Protocol usage donut">
                  {protocolUsage.reduce(
                    (acc, item, index) => {
                      const radius = 70;
                      const circumference = 2 * Math.PI * radius;
                      const total = protocolUsage.reduce((sum, entry) => sum + entry.bytes_total, 0) || 1;
                      const length = (item.bytes_total / total) * circumference;
                      const gap = circumference - length;
                      const dash = `${length} ${gap}`;
                      const element = (
                        <circle
                          key={item.protocol}
                          cx="110"
                          cy="110"
                          r={radius}
                          fill="none"
                          stroke={protocolColors[index % protocolColors.length]}
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
                  {protocolUsage.map((item, index) => (
                    <span key={item.protocol}>
                      <span className="nd-legend-dot" style={{ background: protocolColors[index % protocolColors.length] }} /> {item.protocol}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </Card>

          <Card title="Top Servers by Load">
            {topServers.length === 0 ? (
              <EmptyState title="No data" description="Server load distribution is not available yet." />
            ) : (
              <div className="nd-bars">
                {topServers.map((item) => (
                  <div key={item.node_id} className="nd-bar-row">
                    <strong>{item.hostname}</strong>
                    <ProgressBar value={item.load_percent} hint={`${item.load_percent}%`} />
                    <span>{item.load_percent}%</span>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </section>

        <Card title="Connections by Country">
          <EmptyState title="No data" description="Country-level connections endpoint is not available yet." />
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}

