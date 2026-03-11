import type { User } from "@opener-netdoor/shared-types";
import { hashToPercent } from "../format";

export interface UserRowVM {
  id: string;
  name: string;
  email: string;
  status: "Active" | "Blocked" | "Expired";
  createdAt: string;
  trafficUsedBytes: number;
  trafficLimitBytes: number;
  trafficPercent: number;
  subscription: "Basic" | "Standard" | "Premium" | "Enterprise";
  expiresAt: string;
  dataKind: "backend" | "derived";
}

interface UserNoteMeta {
  display_name?: string;
  subscription?: string;
  traffic_limit_gb?: number;
  traffic_used_bytes?: number;
  expires_at?: string;
}

const PLAN_LIMITS: Record<UserRowVM["subscription"], number> = {
  Basic: 50,
  Standard: 100,
  Premium: 300,
  Enterprise: 1000,
};

function deriveName(email: string, userId: string): string {
  const left = email.split("@")[0]?.trim();
  if (left) {
    return left
      .split(/[._-]/)
      .filter(Boolean)
      .map((token) => token[0].toUpperCase() + token.slice(1))
      .join(" ");
  }
  return `User ${userId.slice(0, 4)}`;
}

function parseUserNote(note?: string): UserNoteMeta {
  if (!note) {
    return {};
  }
  try {
    const parsed = JSON.parse(note) as UserNoteMeta;
    return typeof parsed === "object" && parsed !== null ? parsed : {};
  } catch {
    return {};
  }
}

function normalizeSubscription(raw?: string): UserRowVM["subscription"] | undefined {
  if (!raw) {
    return undefined;
  }
  const value = raw.trim().toLowerCase();
  switch (value) {
    case "basic":
      return "Basic";
    case "standard":
      return "Standard";
    case "premium":
      return "Premium";
    case "enterprise":
      return "Enterprise";
    default:
      return undefined;
  }
}

function derivedSubscription(seed: number): UserRowVM["subscription"] {
  if (seed > 82) {
    return "Enterprise";
  }
  if (seed > 62) {
    return "Premium";
  }
  if (seed > 34) {
    return "Standard";
  }
  return "Basic";
}

function normalizeStatus(status: string, expiresAt?: string): UserRowVM["status"] {
  if (status === "blocked") {
    return "Blocked";
  }
  if (expiresAt) {
    const ts = Date.parse(expiresAt);
    if (Number.isFinite(ts) && ts < Date.now()) {
      return "Expired";
    }
  }
  return "Active";
}

export function toUserRowVM(user: User): UserRowVM {
  const synthetic = hashToPercent(`${user.id}:${user.tenant_id}`);
  const note = parseUserNote(user.note);

  const subscription = normalizeSubscription(note.subscription) ?? derivedSubscription(synthetic);
  const limitGB = typeof note.traffic_limit_gb === "number" && note.traffic_limit_gb > 0 ? note.traffic_limit_gb : PLAN_LIMITS[subscription];

  // Backend does not expose first-class usage/plan/expiry yet, so we consume note metadata when present
  // and fall back to deterministic derived placeholders to keep the table stable.
  const usedBytes =
    typeof note.traffic_used_bytes === "number" && note.traffic_used_bytes >= 0
      ? note.traffic_used_bytes
      : Math.round((limitGB * 1024 * 1024 * 1024 * synthetic) / 100);

  const expiresAt =
    typeof note.expires_at === "string" && note.expires_at.trim()
      ? note.expires_at
      : new Date(Date.now() + (18 + synthetic) * 24 * 60 * 60 * 1000).toISOString();

  const trafficLimitBytes = Math.round(limitGB * 1024 * 1024 * 1024);
  const name = note.display_name?.trim() || deriveName(user.email ?? "", user.id);

  return {
    id: user.id,
    name,
    email: user.email || "unknown@example.com",
    status: normalizeStatus(user.status, expiresAt),
    createdAt: user.created_at,
    trafficUsedBytes: usedBytes,
    trafficLimitBytes,
    trafficPercent: trafficLimitBytes > 0 ? Math.max(0, Math.min(100, (usedBytes / trafficLimitBytes) * 100)) : 0,
    subscription,
    expiresAt,
    dataKind: note.display_name || note.subscription || note.traffic_limit_gb || note.traffic_used_bytes || note.expires_at ? "backend" : "derived",
  };
}

export function toUserRows(users: User[]): UserRowVM[] {
  return users.map(toUserRowVM);
}
