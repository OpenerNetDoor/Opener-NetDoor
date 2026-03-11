"use client";

import Link from "next/link";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { Card, DataTable, PageTitle, SupportBadge } from "../../components/ui";
import { ADVANCED_ITEMS } from "../../lib/nav";

export default function AdvancedPage() {
  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Advanced Tools"
          subtitle="Technical control-plane routes moved out of main owner UX."
          actions={<SupportBadge state="supported" />}
        />

        <Card title="Advanced routes" subtitle="Use only when deep diagnostics or low-level operations are needed.">
          <DataTable
            rows={ADVANCED_ITEMS}
            rowKey={(row) => row.href}
            columns={[
              { id: "name", header: "Route", render: (row) => <Link href={row.href}>{row.label}</Link> },
              { id: "path", header: "Path", render: (row) => <code>{row.href}</code> },
              {
                id: "scope",
                header: "Access",
                render: (row) => (row.platformOnly ? "platform:admin" : row.requiredScopes.join(", ")),
              },
            ]}
          />
        </Card>
      </AdminShell>
    </RouteGuard>
  );
}
