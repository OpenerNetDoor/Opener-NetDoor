import type { Scope } from "@opener-netdoor/shared-types";
import type { AdminSession } from "../auth/session";

export interface RouteAccessRule {
  requiredScopes: Scope[];
  platformOnly?: boolean;
}

export function hasScopes(granted: Scope[], required: Scope[]): boolean {
  return required.every((scope) => granted.includes(scope));
}

export function isPlatformAdmin(scopes: Scope[]): boolean {
  return scopes.includes("platform:admin");
}

export function isTenantScoped(session: Pick<AdminSession, "tenantId" | "scopes">): boolean {
  return Boolean(session.tenantId) && !isPlatformAdmin(session.scopes);
}

export function canAccessRoute(session: AdminSession, rule: RouteAccessRule): boolean {
  if (!hasScopes(session.scopes, rule.requiredScopes)) {
    return false;
  }
  if (rule.platformOnly && isTenantScoped(session)) {
    return false;
  }
  return true;
}

export function resolveTenantForRequest(session: AdminSession, requestedTenantId?: string): string | undefined {
  if (isPlatformAdmin(session.scopes)) {
    return requestedTenantId;
  }
  return session.tenantId ?? requestedTenantId;
}

export type SupportState = "supported" | "frontend_seam" | "planned" | "unsupported";

export function actionSupportLabel(state: SupportState): string {
  switch (state) {
    case "supported":
      return "Backend supported";
    case "frontend_seam":
      return "Frontend seam";
    case "planned":
      return "Planned";
    case "unsupported":
      return "Unsupported";
    default:
      return "Unknown";
  }
}
