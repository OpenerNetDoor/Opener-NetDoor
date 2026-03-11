"use client";

import { useMemo } from "react";
import Link from "next/link";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { ActionButton, Card, PageTitle, StatusBadge } from "../../../components/ui";
import { clearSession, getSession } from "../../../lib/auth/session";

export default function SecuritySettingsPage() {
  const session = useMemo(() => getSession(), []);

  const tokenFingerprint = useMemo(() => {
    if (!session?.token) {
      return "n/a";
    }
    return `${session.token.slice(0, 10)}...${session.token.slice(-6)}`;
  }, [session?.token]);

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle title="Settings · Security" subtitle="Session controls and access visibility." />

        <div className="grid-two">
          <Card title="Session">
            <p>
              token fingerprint <code>{tokenFingerprint}</code>
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
                clearSession();
                window.location.href = "/login";
              }}
            >
              Clear local session
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
