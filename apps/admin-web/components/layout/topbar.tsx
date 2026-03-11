"use client";

import { useEffect, useMemo, useState } from "react";
import type { ThemeMode } from "@opener-netdoor/shared-types";
import {
  Bell,
  LogOut,
  Menu,
  Moon,
  Search,
  Settings,
  Sun,
  UserRound,
} from "lucide-react";
import type { AdminSession } from "../../lib/auth/session";
import { clearSession } from "../../lib/auth/session";
import { formatDateTime, formatRelativeTime } from "../../lib/format";
import type { NotificationView } from "../../lib/mock/notifications";
import { cn } from "../../lib/format";
import { NotificationItem } from "../ui";

interface TopbarProps {
  session: AdminSession | null;
  title: string;
  theme: ThemeMode;
  onThemeChange: (mode: ThemeMode) => void;
  notifications: NotificationView[];
  onOpenMobileMenu: () => void;
}

export function Topbar({ session, title, theme, onThemeChange, notifications, onOpenMobileMenu }: TopbarProps) {
  const [openProfile, setOpenProfile] = useState(false);
  const [openNotifications, setOpenNotifications] = useState(false);
  const [clock, setClock] = useState(() => new Date());

  useEffect(() => {
    const timer = window.setInterval(() => setClock(new Date()), 30_000);
    return () => window.clearInterval(timer);
  }, []);

  const unread = useMemo(() => notifications.filter((item) => !item.read).length, [notifications]);

  return (
    <header className="nd-topbar">
      <div className="nd-topbar-left">
        <button className="nd-top-icon nd-mobile-toggle" onClick={onOpenMobileMenu} type="button" aria-label="Open menu">
          <Menu size={16} />
        </button>
        <label className="nd-search-wrap" aria-label="Search">
          <Search size={16} />
          <input placeholder="Search..." aria-label="Search" />
        </label>
      </div>

      <div className="nd-topbar-right">
        <div className="nd-date-block">
          <span>
            {clock.toLocaleDateString(undefined, {
              weekday: "short",
              month: "short",
              day: "2-digit",
              year: "numeric",
            })}
          </span>
          <small>{clock.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</small>
        </div>

        <button
          className="nd-top-icon"
          type="button"
          onClick={() => onThemeChange(theme === "dark" ? "light" : "dark")}
          aria-label="Toggle theme"
        >
          {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
        </button>

        <div className="nd-popover-wrap">
          <button
            className="nd-top-icon nd-bell-btn"
            type="button"
            onClick={() => {
              setOpenNotifications((prev) => !prev);
              setOpenProfile(false);
            }}
            aria-label="Notifications"
          >
            <Bell size={16} />
            {unread > 0 ? <span className="nd-badge-dot">{unread > 9 ? "9+" : unread}</span> : null}
          </button>

          {openNotifications ? (
            <div className="nd-popover nd-notification-panel">
              <div className="nd-popover-title">Notifications</div>
              {notifications.length === 0 ? (
                <p className="nd-empty-inline">No recent events.</p>
              ) : (
                notifications.slice(0, 8).map((item) => <NotificationItem key={item.id} item={item} />)
              )}
            </div>
          ) : null}
        </div>

        <div className="nd-popover-wrap">
          <button
            className="nd-profile-btn"
            type="button"
            onClick={() => {
              setOpenProfile((prev) => !prev);
              setOpenNotifications(false);
            }}
          >
            <span className="nd-profile-avatar" aria-hidden>
              <UserRound size={14} />
            </span>
            <div className="nd-profile-copy">
              <span>Admin</span>
              <small>{session?.subject ?? "owner"}</small>
            </div>
          </button>

          {openProfile ? (
            <div className="nd-popover nd-profile-panel">
              <p className="nd-popover-title">{title}</p>
              <p className="nd-meta-row">
                Last active: <span>{formatRelativeTime(new Date().toISOString())}</span>
              </p>
              <p className="nd-meta-row">
                Timestamp: <span>{formatDateTime(new Date().toISOString())}</span>
              </p>
              <button className="nd-menu-item" type="button" onClick={() => (window.location.href = "/settings")}> 
                <Settings size={14} /> Profile
              </button>
              <button
                className={cn("nd-menu-item", "danger")}
                type="button"
                onClick={() => {
                  clearSession();
                  window.location.href = "/login";
                }}
              >
                <LogOut size={14} /> Logout
              </button>
            </div>
          ) : null}
        </div>
      </div>
    </header>
  );
}
