import type { Scope } from "@opener-netdoor/shared-types";

export type NavIconKey =
  | "dashboard"
  | "users"
  | "servers"
  | "keys"
  | "subscriptions"
  | "analytics"
  | "settings"
  | "advanced";

export interface NavItem {
  label: string;
  href: string;
  icon: NavIconKey;
  requiredScopes: Scope[];
  platformOnly?: boolean;
  planned?: boolean;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
  hiddenInMain?: boolean;
}

export const NAV_GROUPS: NavGroup[] = [
  {
    label: "Main",
    items: [
      { label: "Dashboard", href: "/dashboard", icon: "dashboard", requiredScopes: ["admin:read"] },
      { label: "Users", href: "/users", icon: "users", requiredScopes: ["admin:read"] },
      { label: "Servers", href: "/servers", icon: "servers", requiredScopes: ["admin:read"] },
      { label: "Keys", href: "/keys", icon: "keys", requiredScopes: ["admin:read"] },
      { label: "Subscriptions", href: "/subscriptions", icon: "subscriptions", requiredScopes: ["admin:read"], planned: true },
      { label: "Analytics", href: "/analytics", icon: "analytics", requiredScopes: ["admin:read"] },
      { label: "Settings", href: "/settings", icon: "settings", requiredScopes: ["admin:read"] },
    ],
  },
  {
    label: "Advanced",
    hiddenInMain: true,
    items: [{ label: "Advanced Tools", href: "/advanced", icon: "advanced", requiredScopes: ["admin:read"] }],
  },
];

export const ADVANCED_ITEMS: NavItem[] = [
  { label: "Tenants", href: "/tenants", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "Tenant Policies", href: "/policies/tenants", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "User Overrides", href: "/policies/users", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "Effective Policy", href: "/policies/effective", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "Provisioning", href: "/nodes/provisioning", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "PKI Issuers", href: "/pki/issuers", icon: "advanced", requiredScopes: ["admin:read"], platformOnly: true },
  { label: "Certificates", href: "/pki/certificates", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "Audit Logs", href: "/audit-logs", icon: "advanced", requiredScopes: ["admin:read"] },
  { label: "Ops Snapshot", href: "/ops-snapshot", icon: "advanced", requiredScopes: ["admin:read"] },
];
