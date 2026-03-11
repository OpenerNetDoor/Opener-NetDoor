"use client";

import Link from "next/link";
import { ChevronLeft, ChevronRight, ShieldCheck } from "lucide-react";
import type { AdminSession } from "../../lib/auth/session";
import { NAV_GROUPS } from "../../lib/nav";
import { canAccessRoute, isPlatformAdmin } from "../../lib/permissions";
import { cn } from "../../lib/format";
import { glyphFor } from "./icon-map";

interface SidebarProps {
  session: AdminSession | null;
  pathname: string;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

export function Sidebar({
  session,
  pathname,
  collapsed,
  onToggleCollapsed,
  mobileOpen,
  onCloseMobile,
}: SidebarProps) {
  const visibleGroups = session
    ? NAV_GROUPS.map((group) => ({
        ...group,
        items: group.items.filter((item) => canAccessRoute(session, item)),
      })).filter((group) => group.items.length > 0)
    : [];

  const mainGroups = visibleGroups.filter((group) => !group.hiddenInMain);

  return (
    <>
      <div className={cn("nd-overlay", mobileOpen && "is-open")} onClick={onCloseMobile} />
      <aside className={cn("nd-sidebar", collapsed && "is-collapsed", mobileOpen && "is-mobile-open")}>
        <div className="nd-brand-wrap">
          <div className="nd-brand-mark" aria-hidden>
            <span className="door" />
          </div>
          {!collapsed ? (
            <div className="nd-brand-copy">
              <strong>Opener</strong>
              <span>NetDoor</span>
            </div>
          ) : null}
          <button className="nd-icon-btn nd-collapse-btn" type="button" onClick={onToggleCollapsed} aria-label="Toggle sidebar width">
            {collapsed ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}
          </button>
        </div>

        <nav className="nd-nav" aria-label="Primary">
          {mainGroups.map((group) => (
            <section key={group.label} className="nd-nav-group">
              <ul>
                {group.items.map((item) => {
                  const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
                  return (
                    <li key={item.href}>
                      <Link href={item.href} className={cn("nd-nav-item", active && "is-active")}> 
                        <span className="nd-nav-icon" aria-hidden>
                          {glyphFor(item.icon)}
                        </span>
                        {!collapsed ? <span className="nd-nav-label">{item.label}</span> : null}
                        {item.planned && !collapsed ? <small className="nd-planned-tag">planned</small> : null}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </nav>

        <footer className="nd-sidebar-footer">
          <div className="nd-role-card">
            <p className="role-title">Admin Access</p>
            <p className="role-meta">Full Permissions</p>
            <p className="role-meta">{session && isPlatformAdmin(session.scopes) ? "Platform owner" : "Single owner scope"}</p>
          </div>
          {!collapsed ? (
            <Link className="nd-advanced-link" href="/advanced">
              <ShieldCheck size={13} style={{ verticalAlign: "text-bottom", marginRight: 6 }} />
              Advanced area
            </Link>
          ) : null}
        </footer>
      </aside>
    </>
  );
}
