"use client";

import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { Card, EmptyState, PageTitle, SupportBadge } from "../../components/ui";

export default function SubscriptionsPage() {
  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Subscriptions"
          subtitle="Billing is not available yet because no production backend billing endpoints are exposed."
          actions={<SupportBadge state="planned" />}
        />

        <Card title="Status" subtitle="This section is intentionally backend-gated.">
          <EmptyState
            title="No data"
            description="Subscription APIs are not implemented yet. This page will activate when billing endpoints are added."
          />
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
