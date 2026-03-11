import type { AuditLogRecord } from "@opener-netdoor/shared-types";
import { formatRelativeTime } from "../format";

export interface AuditEventVM {
  id: string;
  tenant: string;
  action: string;
  actor: string;
  target: string;
  createdAt: string;
  relative: string;
}

export function toAuditEventVM(item: AuditLogRecord): AuditEventVM {
  return {
    id: item.id,
    tenant: item.tenant_id ?? "platform",
    action: item.action,
    actor: item.actor_type,
    target: `${item.target_type ?? "n/a"}:${item.target_id ?? "n/a"}`,
    createdAt: item.created_at,
    relative: formatRelativeTime(item.created_at),
  };
}

export function toAuditEventVMs(items: AuditLogRecord[]): AuditEventVM[] {
  return items.map(toAuditEventVM);
}