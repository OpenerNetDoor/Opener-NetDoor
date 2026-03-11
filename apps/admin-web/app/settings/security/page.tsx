"use client";

import { useMemo } from "react";
import Link from "next/link";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { ActionButton, Card, PageTitle, StatusBadge } from "../../../components/ui";
import { getSession, logoutSession } from "../../../lib/auth/session";

export default function SecuritySettingsPage() {
  const session = useMemo(() => getSession(), []);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle title="Settings · Security" subtitle="Cookie session controls and access visibility." />

        <div className="grid-two">
          <Card title="Session">
            <p>
              Subject: <code>{session?.subject ?? "n/a"}</code>
            </p>
            <p>
              Expires: <code>{session?.expiresAt ?? "n/a"}</code>
            </p>
            <p>Use logout when handing over workstation access.</p>
            <div className="row">
              {(session?.scopes ?? []).map((scope) => (
                <StatusBadge key={scope} value={scope} />
              ))}
            </div>
          </Card>

          <Card title="Controls">
            <ActionButton
              variant="danger"
              onClick={() => {
                void logoutSession(session?.baseUrl).finally(() => {
                  window.location.href = "/login";
                });
              }}
            >
              Logout session
            </ActionButton>
            <div style={{ marginTop: 10 }}>
              <Link href="/advanced">Open advanced diagnostics</Link>
            </div>
          </Card>
        </div>
      </AdminShell>
    </RouteGuard>
  );
}
