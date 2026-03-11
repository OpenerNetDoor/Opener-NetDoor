"use client";

import { useEffect, useMemo, useState } from "react";
import { OpenerNetDoorClient } from "@opener-netdoor/sdk-ts";
import { getSession, hydrateSession, resolveSessionScope, type AdminSession } from "../auth/session";

const defaultBaseURL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

export function useAdminSession(): AdminSession | null {
  const [session, setSession] = useState<AdminSession | null>(null);

  useEffect(() => {
    const local = getSession();
    if (local) {
      setSession(local);
    }

    void hydrateSession(local?.baseUrl ?? defaultBaseURL)
      .then((next) => {
        if (next) {
          setSession(next);
        }
      })
      .catch(() => {
        if (!local) {
          setSession(null);
        }
      });

    const onStorage = () => setSession(getSession());
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  return session;
}

export function useOwnerScopeId(): string | undefined {
  const session = useAdminSession();
  return useMemo(() => resolveSessionScope(session), [session]);
}

export function useAPIClient() {
  const session = useAdminSession();
  const scopeId = useMemo(() => resolveSessionScope(session), [session]);

  return useMemo(
    () =>
      new OpenerNetDoorClient({
        baseUrl: session?.baseUrl ?? defaultBaseURL,
        tenantId: scopeId,
      }),
    [scopeId, session?.baseUrl],
  );
}

export function ensureBaseURL(baseURL: string): string {
  const normalized = baseURL.trim().replace(/\/+$/, "");
  if (!normalized.startsWith("http://") && !normalized.startsWith("https://")) {
    return `http://${normalized}`;
  }
  return normalized;
}

