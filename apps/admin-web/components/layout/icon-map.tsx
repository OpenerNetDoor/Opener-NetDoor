import type { ReactNode } from "react";
import {
  BarChart3,
  CreditCard,
  Gauge,
  KeyRound,
  Server,
  Settings,
  Shield,
  Users,
} from "lucide-react";

export type NavGlyphKey =
  | "dashboard"
  | "users"
  | "servers"
  | "keys"
  | "subscriptions"
  | "analytics"
  | "settings"
  | "advanced";

const GLYPH_MAP: Record<NavGlyphKey, ReactNode> = {
  dashboard: <Gauge size={16} strokeWidth={2} />,
  users: <Users size={16} strokeWidth={2} />,
  servers: <Server size={16} strokeWidth={2} />,
  keys: <KeyRound size={16} strokeWidth={2} />,
  subscriptions: <CreditCard size={16} strokeWidth={2} />,
  analytics: <BarChart3 size={16} strokeWidth={2} />,
  settings: <Settings size={16} strokeWidth={2} />,
  advanced: <Shield size={16} strokeWidth={2} />,
};

export function glyphFor(key: NavGlyphKey): ReactNode {
  return GLYPH_MAP[key] ?? <Gauge size={16} strokeWidth={2} />;
}
