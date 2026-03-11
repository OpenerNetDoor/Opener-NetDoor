"use client";

import { useParams, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { EffectivePolicy, User } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import { Card, ErrorState, LoadingState, PageTitle, StatusBadge } from "../../../components/ui";
import { useAPIClient, useOwnerScopeId } from "../../../lib/api/client";

export default function UserDetailPage() {
  const params = useParams<{ userId: string }>();
  const search = useSearchParams();
  const api = useAPIClient();
  const ownerScopeId = useOwnerScopeId();

  const userId = useMemo(() => String(params.userId ?? ""), [params]);
  const tenantId = search.get("tenant_id") ?? ownerScopeId ?? "";

  const [user, setUser] = useState<User | null>(null);
  const [effectivePolicy, setEffectivePolicy] = useState<EffectivePolicy | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId || !userId) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const users = await api.listUsersPage({ tenantId, limit: 100, offset: 0 });
        const found = users.items.find((item) => item.id === userId) ?? null;
        const policy = await api.getEffectivePolicy(tenantId, userId);
        if (cancelled) {
          return;
        }
        setUser(found);
        setEffectivePolicy(policy);
        setError("");
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "request failed");
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [api, tenantId, userId]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle title="User Detail" subtitle="User profile and policy view." />

        <Card title="Context">
          <p>
            user id <code>{userId}</code>
          </p>
          <p>
            workspace scope <code>{tenantId || "n/a"}</code>
          </p>
        </Card>

        {error ? <ErrorState message={error} /> : null}
        {!error && !user ? <LoadingState label="Loading user..." /> : null}

        {user ? (
          <Card title="Profile">
            <p>
              <strong>{user.email || "n/a"}</strong>
            </p>
            <p>
              Status: <StatusBadge value={user.status} />
            </p>
            <p>Created: {user.created_at}</p>
            <p>
              Internal id: <code>{user.id}</code>
            </p>
          </Card>
        ) : null}

        <Card title="Effective policy">
          {effectivePolicy ? (
            <pre>{JSON.stringify(effectivePolicy, null, 2)}</pre>
          ) : (
            <p>No effective policy was resolved for this user.</p>
          )}
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
