import type { User } from "@opener-netdoor/shared-types";

export interface UserRowVM {
  id: string;
  name: string;
  email: string;
  status: "Active" | "Blocked" | "Expired";
  createdAt: string;
  trafficUsedBytes: number | null;
  trafficLimitBytes: number | null;
  trafficPercent: number | null;
  subscription: "Basic" | "Standard" | "Premium" | "Enterprise" | null;
  expiresAt: string | null;
  dataKind: "backend" | "empty";
}

interface UserNoteMeta {
  display_name?: string;
  subscription?: string;
  traffic_limit_gb?: number;
  traffic_used_bytes?: number;
  expires_at?: string;
}

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
  const note = parseUserNote(user.note);

  const subscription = normalizeSubscription(note.subscription) ?? null;
  const trafficLimitBytes =
    typeof note.traffic_limit_gb === "number" && note.traffic_limit_gb > 0
      ? Math.round(note.traffic_limit_gb * 1024 * 1024 * 1024)
      : null;

  const trafficUsedBytes =
    typeof note.traffic_used_bytes === "number" && note.traffic_used_bytes >= 0 ? note.traffic_used_bytes : null;

  const expiresAt = typeof note.expires_at === "string" && note.expires_at.trim() ? note.expires_at : null;
  const name = note.display_name?.trim() || deriveName(user.email ?? "", user.id);

  return {
    id: user.id,
    name,
    email: user.email || "unknown@example.com",
    status: normalizeStatus(user.status, expiresAt ?? undefined),
    createdAt: user.created_at,
    trafficUsedBytes,
    trafficLimitBytes,
    trafficPercent:
      typeof trafficUsedBytes === "number" && typeof trafficLimitBytes === "number" && trafficLimitBytes > 0
        ? Math.max(0, Math.min(100, (trafficUsedBytes / trafficLimitBytes) * 100))
        : null,
    subscription,
    expiresAt,
    dataKind:
      note.display_name || note.subscription || note.traffic_limit_gb || note.traffic_used_bytes || note.expires_at
        ? "backend"
        : "empty",
  };
}

export function toUserRows(users: User[]): UserRowVM[] {
  return users.map(toUserRowVM);
}
