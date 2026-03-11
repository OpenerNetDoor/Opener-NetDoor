"use client";

import { useEffect, useMemo, useState } from "react";
import type {
  NodeProvisioningContract,
  ServerInventoryItem,
  ServerOperationJob,
  ServerOperationPlan,
  ServerOperationType,
} from "@opener-netdoor/shared-types";
import { AdminShell } from "../../../components/admin-shell";
import { RouteGuard } from "../../../components/route-guard";
import {
  Card,
  CommandPreview,
  DataTable,
  ErrorState,
  LogViewer,
  PageTitle,
  StatusBadge,
  StepperWizard,
  SupportBadge,
} from "../../../components/ui";
import { useAPIClient, useAdminSession } from "../../../lib/api/client";
import { getServerOperationsAdapter } from "../../../lib/server-ops";

const STEPS = [
  { id: "preflight", label: "Preflight checks" },
  { id: "plan", label: "Install plan" },
  { id: "execute", label: "Execute operation" },
  { id: "verify", label: "Ready + certificate verify" },
];

const OPERATION_OPTIONS: ServerOperationType[] = [
  "install",
  "upgrade",
  "rollback",
  "backup",
  "restore",
  "cert_rotate",
  "firewall_apply",
  "cdn_apply",
];

export default function NodeProvisioningPage() {
  const api = useAPIClient();
  const session = useAdminSession();
  const adapter = useMemo(() => getServerOperationsAdapter(), []);

  const [tenantId, setTenantId] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [nodeKeyId, setNodeKeyId] = useState("");
  const [provisioning, setProvisioning] = useState<NodeProvisioningContract | null>(null);

  const [servers, setServers] = useState<ServerInventoryItem[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [operation, setOperation] = useState<ServerOperationType>("install");
  const [plan, setPlan] = useState<ServerOperationPlan | null>(null);
  const [jobs, setJobs] = useState<ServerOperationJob[]>([]);

  const [step, setStep] = useState("preflight");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!tenantId && session?.tenantId) {
      setTenantId(session.tenantId);
    }
  }, [session?.tenantId, tenantId]);

  useEffect(() => {
    void adapter
      .listServers()
      .then((items) => {
        setServers(items);
        if (items[0] && !selectedServer) {
          setSelectedServer(items[0].id);
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : "server inventory unavailable"));

    void adapter
      .listJobs()
      .then(setJobs)
      .catch((err) => setError(err instanceof Error ? err.message : "job history unavailable"));
  }, [adapter]);

  return (
    <RouteGuard requiredScopes={["admin:read"]} expectedTenantId={tenantId || undefined}>
      <AdminShell>
        <PageTitle
          title="Node Provisioning"
          subtitle="VDS/server workflow UX with explicit local planner seam where backend orchestration is not exposed."
          actions={<SupportBadge state={adapter.source === "local_planner" ? "frontend_seam" : "supported"} />}
        />

        <Card title="Provisioning contract lookup" subtitle="Backed by real endpoint /v1/admin/nodes/provisioning.">
          <div className="row">
            <input placeholder="tenant_id" value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
            <input placeholder="node_id (optional)" value={nodeId} onChange={(event) => setNodeId(event.target.value)} />
            <input placeholder="node_key_id (optional)" value={nodeKeyId} onChange={(event) => setNodeKeyId(event.target.value)} />
            <button
              className="nd-btn is-primary"
              onClick={() => {
                if (!tenantId.trim()) {
                  setError("tenant_id is required");
                  return;
                }
                setStep("plan");
                void api
                  .getNodeProvisioning({
                    tenantId: tenantId.trim(),
                    nodeId: nodeId.trim() || undefined,
                    nodeKeyId: nodeKeyId.trim() || undefined,
                  })
                  .then((result) => {
                    setProvisioning(result);
                    setError("");
                    setStep("verify");
                  })
                  .catch((err) => {
                    setError(err instanceof Error ? err.message : "contract lookup failed");
                    setStep("preflight");
                  });
              }}
            >
              Resolve contract
            </button>
          </div>
          {provisioning ? <pre>{JSON.stringify(provisioning, null, 2)}</pre> : <p>No contract loaded yet.</p>}
        </Card>

        <Card title="Operations center" subtitle="Install/upgrade/rollback/backup planner mapped from reference UX.">
          <StepperWizard steps={STEPS} current={step} />

          <div className="row" style={{ marginTop: 12 }}>
            <select value={selectedServer} onChange={(event) => setSelectedServer(event.target.value)}>
              {servers.map((server) => (
                <option key={server.id} value={server.id}>
                  {server.id} ({server.region})
                </option>
              ))}
            </select>
            <select value={operation} onChange={(event) => setOperation(event.target.value as ServerOperationType)}>
              {OPERATION_OPTIONS.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
            <button
              className="nd-btn is-secondary"
              onClick={() => {
                if (!selectedServer) {
                  setError("select server first");
                  return;
                }
                setStep("plan");
                void adapter
                  .createPlan({ serverId: selectedServer, operation })
                  .then((result) => {
                    setPlan(result);
                    setError("");
                  })
                  .catch((err) => setError(err instanceof Error ? err.message : "plan generation failed"));
              }}
            >
              Generate plan
            </button>
            <button
              className="nd-btn is-primary"
              onClick={() => {
                if (!selectedServer) {
                  setError("select server first");
                  return;
                }
                setStep("execute");
                void adapter
                  .startOperation({ serverId: selectedServer, operation })
                  .then(() => adapter.listJobs())
                  .then((result) => {
                    setJobs(result);
                    setStep("verify");
                    setError("");
                  })
                  .catch((err) => setError(err instanceof Error ? err.message : "operation launch failed"));
              }}
            >
              Start operation
            </button>
          </div>

          {plan ? <CommandPreview title={`${plan.operation} command preview`} commands={plan.commands} warnings={plan.warnings} /> : null}
        </Card>

        <Card title="Server inventory">
          <DataTable
            rows={servers}
            rowKey={(row) => row.id}
            columns={[
              { id: "id", header: "Server", render: (row) => <code>{row.id}</code> },
              { id: "region", header: "Region", render: (row) => row.region },
              { id: "ip", header: "IP", render: (row) => row.ip },
              { id: "domain", header: "Domain", render: (row) => row.domain ?? "n/a" },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "source", header: "Source", render: () => <SupportBadge state={adapter.source === "local_planner" ? "frontend_seam" : "supported"} /> },
            ]}
          />
        </Card>

        <Card title="Job history and logs">
          <DataTable
            rows={jobs}
            rowKey={(row) => row.id}
            columns={[
              { id: "job", header: "Job", render: (row) => <code>{row.id}</code> },
              { id: "server", header: "Server", render: (row) => row.server_id },
              { id: "operation", header: "Operation", render: (row) => row.operation },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              { id: "corr", header: "Correlation", render: (row) => <code>{row.correlation_id}</code> },
            ]}
          />
          <LogViewer logs={jobs.flatMap((job) => job.logs)} />
        </Card>

        {error ? <ErrorState message={error} /> : null}
      </AdminShell>
    </RouteGuard>
  );
}