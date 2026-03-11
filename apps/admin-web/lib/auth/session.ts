"use client";

import type { Scope, ThemeMode } from "@opener-netdoor/shared-types";

const SESSION_KEY = "opener-netdoor-admin-session";
const THEME_KEY = "opener-netdoor-admin-theme";
const DEFAULT_SCOPE_KEY = "opener-netdoor-admin-default-scope";
const DEFAULT_SCOPE_FALLBACK = process.env.NEXT_PUBLIC_DEFAULT_SCOPE_ID ?? "default";
const SINGLE_OWNER_MODE = (process.env.NEXT_PUBLIC_SINGLE_OWNER_MODE ?? "true") !== "false";

export interface AdminSession {
  subject: string;
  token: string;
  tenantId?: string;
  scopes: Scope[];
  baseUrl: string;
  theme?: ThemeMode;
}

export function getSession(): AdminSession | null {
  if (typeof window === "undefined") {
    return null;
  }
  const raw = window.localStorage.getItem(SESSION_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as AdminSession;
    const effectiveScope = parsed.tenantId?.trim() || resolveDefaultScopeId();
    return {
      ...parsed,
      tenantId: effectiveScope,
    };
  } catch {
    return null;
  }
}

export function setSession(session: AdminSession): void {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(
    SESSION_KEY,
    JSON.stringify({
      ...session,
      baseUrl: session.baseUrl.trim().replace(/\/+$/, ""),
      tenantId: session.tenantId?.trim() || resolveDefaultScopeId(),
      subject: session.subject.trim(),
      token: session.token.trim(),
      scopes: Array.from(new Set(session.scopes)),
    }),
  );
}

export function clearSession(): void {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.removeItem(SESSION_KEY);
}

export function setTheme(theme: ThemeMode): void {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(THEME_KEY, theme);
}

export function getTheme(): ThemeMode {
  if (typeof window === "undefined") {
    return "dark";
  }
  const value = window.localStorage.getItem(THEME_KEY);
  if (value === "light" || value === "dark" || value === "system") {
    return value;
  }
  return "dark";
}

export function isSingleOwnerMode(): boolean {
  return SINGLE_OWNER_MODE;
}

export function resolveDefaultScopeId(): string | undefined {
  if (!SINGLE_OWNER_MODE) {
    return undefined;
  }
  if (typeof window === "undefined") {
    return DEFAULT_SCOPE_FALLBACK;
  }
  const stored = window.localStorage.getItem(DEFAULT_SCOPE_KEY)?.trim();
  if (stored) {
    return stored;
  }
  window.localStorage.setItem(DEFAULT_SCOPE_KEY, DEFAULT_SCOPE_FALLBACK);
  return DEFAULT_SCOPE_FALLBACK;
}

export function setDefaultScopeId(scopeId: string): void {
  if (typeof window === "undefined") {
    return;
  }
  const normalized = scopeId.trim();
  if (!normalized) {
    return;
  }
  window.localStorage.setItem(DEFAULT_SCOPE_KEY, normalized);
}

export function resolveSessionScope(session: AdminSession | null | undefined): string | undefined {
  if (session?.tenantId?.trim()) {
    return session.tenantId.trim();
  }
  return resolveDefaultScopeId();
}
