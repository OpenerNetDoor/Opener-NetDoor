import type { LucideIcon } from "lucide-react";
import { Activity, KeyRound, Server, Users } from "lucide-react";
import type { OpsAnalytics } from "@opener-netdoor/shared-types";
import { formatBytes } from "../format";

export interface DashboardCardVM {
  id: string;
  label: string;
  value: string;
  delta?: string;
  tone: "neutral" | "success" | "warning" | "danger";
  icon: LucideIcon;
}

export interface DashboardQuickAction {
  id: string;
  label: string;
  href: string;
  support: "supported" | "frontend_seam" | "planned" | "unsupported";
}

export function buildDashboardCards(input: {
  fallbackUserCount: number;
  fallbackNodeCount: number;
  analytics?: OpsAnalytics | null;
}): DashboardCardVM[] {
  const activeUsers = input.analytics?.active_users ?? input.fallbackUserCount;
  const activeKeys = input.analytics?.active_keys ?? 0;
  const onlineServers = input.analytics?.online_servers ?? 0;

  return [
    {
      id: "users",
      label: "Active Users",
      value: String(activeUsers),
      tone: "success",
      icon: Users,
    },
    {
      id: "traffic",
      label: "Traffic (24h)",
      value: formatBytes(input.analytics?.traffic_bytes_24h ?? 0),
      tone: "success",
      icon: Activity,
    },
    {
      id: "servers",
      label: "Servers Online",
      value: `${onlineServers}/${input.fallbackNodeCount}`,
      tone: onlineServers > 0 ? "success" : "neutral",
      icon: Server,
    },
    {
      id: "keys",
      label: "Active Keys",
      value: String(activeKeys),
      tone: activeKeys > 0 ? "success" : "neutral",
      icon: KeyRound,
    },
  ];
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
