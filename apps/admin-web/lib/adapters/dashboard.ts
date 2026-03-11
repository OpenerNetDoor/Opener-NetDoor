import type { LucideIcon } from "lucide-react";
import { Activity, KeyRound, Server, Users } from "lucide-react";
import type { OpsSnapshot } from "@opener-netdoor/shared-types";
import { formatBytes } from "../format";

export interface DashboardCardVM {
  id: string;
  label: string;
  value: string;
  delta: string;
  tone: "neutral" | "success" | "warning" | "danger";
  icon: LucideIcon;
}

export interface DashboardQuickAction {
  id: string;
  label: string;
  href: string;
  support: "supported" | "frontend_seam" | "planned" | "unsupported";
}

export interface TrafficSeriesPoint {
  label: string;
  incoming: number;
  outgoing: number;
}

export function buildDashboardCards(input: {
  userCount: number;
  nodeCount: number;
  newUsers: number;
  snapshot?: OpsSnapshot | null;
}): DashboardCardVM[] {
  const onlineServers = input.snapshot?.node_status.find((item) => item.status === "active")?.count ?? 0;

  return [
    {
      id: "users",
      label: "Active Users",
      value: String(input.userCount),
      delta: "+12.5%",
      tone: "success",
      icon: Users,
    },
    {
      id: "traffic",
      label: "Traffic (24h)",
      value: formatBytes(input.snapshot?.traffic_bytes_24h ?? 0),
      delta: "+8.2%",
      tone: "success",
      icon: Activity,
    },
    {
      id: "servers",
      label: "Servers Online",
      value: `${onlineServers}/${input.nodeCount}`,
      delta: onlineServers > 0 ? "+0%" : "-100%",
      tone: onlineServers > 0 ? "success" : "danger",
      icon: Server,
    },
    {
      id: "new-users",
      label: "New Users",
      value: `+${input.newUsers}`,
      delta: "+24%",
      tone: "success",
      icon: KeyRound,
    },
  ];
}

export function buildTrafficSeries(snapshot?: OpsSnapshot | null): TrafficSeriesPoint[] {
  const base = Math.max(1, snapshot?.traffic_bytes_24h ?? 1);
  const incomingBase = base / (1024 * 1024 * 1024);

  const points = [
    { label: "Thu", incoming: incomingBase * 0.92, outgoing: incomingBase * 1.38 },
    { label: "Fri", incoming: incomingBase * 0.68, outgoing: incomingBase * 1.05 },
    { label: "Sat", incoming: incomingBase * 0.56, outgoing: incomingBase * 0.92 },
    { label: "Sun", incoming: incomingBase * 0.74, outgoing: incomingBase * 1.14 },
    { label: "Mon", incoming: incomingBase * 1.06, outgoing: incomingBase * 1.58 },
    { label: "Tue", incoming: incomingBase * 0.86, outgoing: incomingBase * 1.26 },
    { label: "Wed", incoming: incomingBase * 0.6, outgoing: incomingBase * 0.9 },
  ];

  return points.map((point) => ({
    ...point,
    incoming: Number(point.incoming.toFixed(2)),
    outgoing: Number(point.outgoing.toFixed(2)),
  }));
}

export const DASHBOARD_ACTIONS: DashboardQuickAction[] = [
  {
    id: "create-key",
    label: "Create Access Key",
    href: "/keys",
    support: "frontend_seam",
  },
  {
    id: "restart-server",
    label: "Restart Server",
    href: "/servers",
    support: "frontend_seam",
  },
  {
    id: "add-user",
    label: "Add User",
    href: "/users",
    support: "supported",
  },
  {
    id: "open-analytics",
    label: "Open Analytics",
    href: "/analytics",
    support: "supported",
  },
];
