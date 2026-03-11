const SESSION_KEY = "opener-netdoor-manager-session";

export interface ManagerSession {
  baseUrl: string;
  token: string;
  tenantId?: string;
  subject?: string;
  scopes: string[];
  theme: "dark" | "light";
}

export function saveSession(input: {
  baseUrl: string;
  token: string;
  tenantId?: string;
  theme?: "dark" | "light";
}): ManagerSession {
  const payload = decodeJWTPayload(input.token);
  const scopes = Array.isArray(payload?.scopes)
    ? payload!.scopes.filter((item): item is string => typeof item === "string")
    : ["admin:read"];

  const normalized: ManagerSession = {
    baseUrl: input.baseUrl.trim().replace(/\/+$/, ""),
    token: input.token.trim(),
    tenantId: input.tenantId?.trim() || payload?.tenant_id || undefined,
    subject: typeof payload?.sub === "string" ? payload.sub : undefined,
    scopes,
    theme: input.theme ?? "dark",
  };

  window.localStorage.setItem(SESSION_KEY, JSON.stringify(normalized));
  return normalized;
}

export function getSession(): ManagerSession | null {
  const raw = window.localStorage.getItem(SESSION_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<ManagerSession>;
    if (!parsed.baseUrl || !parsed.token) {
      return null;
    }
    return {
      baseUrl: parsed.baseUrl,
      token: parsed.token,
      tenantId: parsed.tenantId,
      subject: parsed.subject,
      scopes: Array.isArray(parsed.scopes) ? parsed.scopes : ["admin:read"],
      theme: parsed.theme === "light" ? "light" : "dark",
    };
  } catch {
    return null;
  }
}

export function clearSession(): void {
  window.localStorage.removeItem(SESSION_KEY);
}

export function hasScope(session: ManagerSession | null, scope: string): boolean {
  return Boolean(session?.scopes.includes(scope));
}

function decodeJWTPayload(token: string): Record<string, unknown> | null {
  const parts = token.split(".");
  if (parts.length < 2) {
    return null;
  }
  try {
    const normalized = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const decoded = atob(normalized);
    return JSON.parse(decoded) as Record<string, unknown>;
  } catch {
    return null;
  }
}
