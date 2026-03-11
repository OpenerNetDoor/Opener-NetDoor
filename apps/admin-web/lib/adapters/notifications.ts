import type { AuditLogRecord } from "@opener-netdoor/shared-types";
import { formatRelativeTime } from "../format";

export interface NotificationView {
  id: string;
  title: string;
  subtitle: string;
  when: string;
  tone: "success" | "warning" | "danger" | "neutral";
  read?: boolean;
}

function toneFromAction(action: string): NotificationView["tone"] {
  const normalized = action.toLowerCase();
  if (normalized.includes("revoke") || normalized.includes("reject") || normalized.includes("deny") || normalized.includes("fail")) {
    return "danger";
  }
  if (normalized.includes("expire") || normalized.includes("renew") || normalized.includes("rotate")) {
    return "warning";
  }
  if (normalized.includes("create") || normalized.includes("accept") || normalized.includes("reactivate") || normalized.includes("connect")) {
    return "success";
  }
  return "neutral";
}

export function buildNotificationFeed(logs: AuditLogRecord[]): NotificationView[] {
  return logs.map((item) => ({
    id: item.id,
    title: item.action,
    subtitle: [item.target_type, item.target_id].filter(Boolean).join(" ") || "Audit event",
    when: formatRelativeTime(item.created_at),
    tone: toneFromAction(item.action),
    read: false,
  }));
}
