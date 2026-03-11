"use client";

import { useEffect, useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import type { ThemeMode } from "@opener-netdoor/shared-types";
import { OpenerNetDoorClient } from "@opener-netdoor/sdk-ts";
import { getSession, getTheme, resolveSessionScope, setTheme, type AdminSession } from "../../lib/auth/session";
import { buildNotificationFeed, type NotificationView } from "../../lib/adapters/notifications";
import { Sidebar } from "./sidebar";
import { Topbar } from "./topbar";

const TITLE_MAP: Record<string, string> = {
  dashboard: "Dashboard",
  users: "Users",
  servers: "Servers",
  nodes: "Servers",
  keys: "Keys",
  "access-keys": "Keys",
  subscriptions: "Subscriptions",
  analytics: "Analytics",
  settings: "Settings",
  advanced: "Advanced",
};

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [session, setSession] = useState<AdminSession | null>(null);
  const [theme, setThemeState] = useState<ThemeMode>("dark");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileSidebar, setMobileSidebar] = useState(false);
  const [notifications, setNotifications] = useState<NotificationView[]>([]);

  useEffect(() => {
    const next = getSession();
    setSession(next);
    setThemeState(getTheme());

    const onStorage = () => {
      setSession(getSession());
      setThemeState(getTheme());
    };

    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  useEffect(() => {
    setMobileSidebar(false);
  }, [pathname]);

  useEffect(() => {
    const root = document.documentElement;
    if (theme === "system") {
      const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      root.dataset.theme = dark ? "dark" : "light";
      return;
    }
    root.dataset.theme = theme;
  }, [theme]);

  useEffect(() => {
    let cancelled = false;

    if (!session?.baseUrl) {
      setNotifications([]);
      return () => {
        cancelled = true;
      };
    }

    const scopeId = resolveSessionScope(session);
    const client = new OpenerNetDoorClient({
      baseUrl: session.baseUrl,
      tenantId: scopeId,
    });

    void client
      .listAuditLogs({ tenantId: scopeId, limit: 8, offset: 0 })
      .then((res) => {
        if (!cancelled) {
          setNotifications(buildNotificationFeed(res.items));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setNotifications([]);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [pathname, session?.baseUrl, session?.tenantId]);

  const title = useMemo(() => {
    const parts = pathname.split("/").filter(Boolean);
    if (parts.length === 0) {
      return "Dashboard";
    }
    const first = parts[0];
    return TITLE_MAP[first] ?? parts.map((part) => part.replace(/-/g, " ")).join(" / ");
  }, [pathname]);

  return (
    <div className="nd-shell">
      <Sidebar
        session={session}
        pathname={pathname}
        collapsed={sidebarCollapsed}
        onToggleCollapsed={() => setSidebarCollapsed((prev) => !prev)}
        mobileOpen={mobileSidebar}
        onCloseMobile={() => setMobileSidebar(false)}
      />
      <div className="nd-main">
        <Topbar
          session={session}
          title={title}
          theme={theme}
          onThemeChange={(next) => {
            setTheme(next);
            setThemeState(next);
          }}
          notifications={notifications}
          onOpenMobileMenu={() => setMobileSidebar(true)}
        />
        <main className="nd-content">{children}</main>
      </div>
    </div>
  );
}


