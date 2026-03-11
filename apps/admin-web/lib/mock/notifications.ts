import { formatRelativeTime } from "../format";

export interface NotificationView {
  id: string;
  title: string;
  subtitle: string;
  when: string;
  tone: "success" | "warning" | "danger" | "neutral";
  read?: boolean;
}

export function buildHeaderNotifications(pathname: string): NotificationView[] {
  const now = Date.now();
  const base = [
    {
      id: "notif-server-ok",
      title: "Server heartbeat received",
      subtitle: "Latest server check-in is healthy",
      ts: now - 2 * 60_000,
      tone: "success" as const,
      read: false,
    },
    {
      id: "notif-cert-expiry",
      title: "Certificate renewal window",
      subtitle: "At least one certificate expires in less than 24h",
      ts: now - 11 * 60_000,
      tone: "warning" as const,
      read: false,
    },
    {
      id: "notif-navigation",
      title: `Opened ${pathname}`,
      subtitle: "Panel navigation event",
      ts: now - 31 * 60_000,
      tone: "neutral" as const,
      read: true,
    },
    {
      id: "notif-security",
      title: "Suspicious request blocked",
      subtitle: "Replay protection denied a duplicate signed request",
      ts: now - 75 * 60_000,
      tone: "danger" as const,
      read: true,
    },
  ];

  return base.map((item) => ({
    id: item.id,
    title: item.title,
    subtitle: item.subtitle,
    when: formatRelativeTime(new Date(item.ts).toISOString()),
    tone: item.tone,
    read: item.read,
  }));
}
