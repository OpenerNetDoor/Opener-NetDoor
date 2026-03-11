"use client";

import { useEffect, useMemo, useState } from "react";
import type { ThemeMode } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, PageTitle, StatusBadge } from "../../../components/ui";
import { getSession, getTheme, setTheme } from "../../../lib/auth/session";

export default function ProfileSettingsPage() {
  const [subject, setSubject] = useState("");
  const [theme, setThemeMode] = useState<ThemeMode>("dark");

  useEffect(() => {
    const session = getSession();
    setSubject(session?.subject ?? "owner");
    setThemeMode(getTheme());
  }, []);

  const scopeTags = useMemo(() => getSession()?.scopes ?? [], []);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle title="Settings · Profile" subtitle="Owner profile and panel appearance preferences." />

        <div className="grid-two">
          <Card title="Owner profile">
            <p>
              Signed in as <strong>{subject || "owner"}</strong>
            </p>
            <p>This panel runs in single-owner mode with hidden scope routing.</p>
            <div className="row">
              {scopeTags.map((scope) => (
                <StatusBadge key={scope} value={scope} />
              ))}
            </div>
          </Card>

          <Card title="Appearance">
            <div className="row" style={{ maxWidth: 360 }}>
              <select
                value={theme}
                onChange={(event) => {
                  const next = event.target.value as ThemeMode;
                  setTheme(next);
                  setThemeMode(next);
                }}
              >
                <option value="dark">dark</option>
                <option value="light">light</option>
                <option value="system">system</option>
              </select>
            </div>
            <p>Theme is stored locally for this browser profile.</p>
          </Card>
        </div>
      </AdminShell>
    </RouteGuard>
  );
}
