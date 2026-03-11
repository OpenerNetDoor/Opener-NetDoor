"use client";

import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { Card, PageTitle, SupportBadge } from "../../components/ui";
import { SUBSCRIPTION_PLANNED_ACTIONS } from "../../lib/mock/planned";

export default function SubscriptionsPage() {
  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Subscriptions"
          subtitle="Billing and plan controls are prepared as a product seam and intentionally not faked without backend endpoints."
          actions={<SupportBadge state="planned" />}
        />

        <Card title="Current stage" subtitle="No production billing endpoints are exposed in Stage 1–12 backend runtime.">
          <div style={{ display: "grid", gap: 10 }}>
            {SUBSCRIPTION_PLANNED_ACTIONS.map((item) => (
              <article key={item.id} className="nd-state">
                <div className="row">
                  <strong>{item.label}</strong>
                  <SupportBadge state={item.support} />
                </div>
                <p>{item.note}</p>
              </article>
            ))}
          </div>
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
