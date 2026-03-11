"use client";

import { useEffect, useState } from "react";
import type { Scope } from "@opener-netdoor/shared-types";
import { getSession, type AdminSession } from "../lib/auth/session";
import { hasScopes, isTenantScoped } from "../lib/permissions";
import { LoadingState, PermissionDeniedState, ScopeMismatchState } from "./ui";

export function RouteGuard({
  requiredScopes,
  children,
  platformOnly,
  expectedTenantId,
}: {
  requiredScopes: Scope[];
  children: React.ReactNode;
  platformOnly?: boolean;
  expectedTenantId?: string;
}) {
  const [session, setSession] = useState<AdminSession | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const next = getSession();
    if (!next) {
      window.location.href = "/login";
      return;
    }
    setSession(next);
    setReady(true);
  }, []);

  if (!ready || !session) {
    return <LoadingState label="Loading session..." />;
  }

  if (!hasScopes(session.scopes, requiredScopes)) {
    return <PermissionDeniedState />;
  }

  if (platformOnly && isTenantScoped(session)) {
    return <ScopeMismatchState expected={expectedTenantId} actual={session.tenantId} />;
  }

  if (expectedTenantId && session.tenantId && session.tenantId !== expectedTenantId && isTenantScoped(session)) {
    return <ScopeMismatchState expected={expectedTenantId} actual={session.tenantId} />;
  }

  return <>{children}</>;
}
