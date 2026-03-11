"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Activity,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Trash2,
  Zap,
} from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  ActionButton,
  DrawerShell,
  EmptyState,
  ErrorState,
  LoadingState,
  ModalShell,
  PageTitle,
  ProgressBar,
  ProtocolChip,
  StatCard,
  StatusBadge,
  SupportBadge,
} from "../../components/ui";
import { useAPIClient, useAdminSession, useOwnerScopeId } from "../../lib/api/client";
import { toNodeRows, type NodeRowVM } from "../../lib/adapters/nodes";
import { hasScopes } from "../../lib/permissions";
import { emitAdminDataChanged } from "../../lib/events/admin-data";

interface AddServerForm {
  hostname: string;
  region: string;
  agentVersion: string;
  capabilities: string[];
}

const DEFAULT_FORM: AddServerForm = {
  hostname: "",
  region: "us-central",
  agentVersion: "owner-manual",
  capabilities: ["heartbeat.v1", "provisioning.v1"],
};

const CAPABILITY_OPTIONS = ["heartbeat.v1", "provisioning.v1", "metrics.v1", "diagnostics.v1"];

export default function ServersPage() {
  const api = useAPIClient();
  const session = useAdminSession();
  const scopeId = useOwnerScopeId();

  const [status, setStatus] = useState("");
  const [items, setItems] = useState<NodeRowVM[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [selected, setSelected] = useState<NodeRowVM | null>(null);
  const [detail, setDetail] = useState<NodeRowVM | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const [showAddServer, setShowAddServer] = useState(false);
  const [createForm, setCreateForm] = useState<AddServerForm>(DEFAULT_FORM);
  const [createPending, setCreatePending] = useState(false);

  const [rowPendingId, setRowPendingId] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  const load = async () => {
    if (!scopeId) {
      setError("Owner scope is not available in session");
      return;
    }
    setLoading(true);
    try {
      const result = await api.listNodesPage({
        tenantId: scopeId,
        status: status || undefined,
        limit: 200,
        offset: 0,
      });
      setItems(toNodeRows(result.items));
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "request failed");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [scopeId, status]);

  useEffect(() => {
    if (!selected || !scopeId) {
      setDetail(null);
      return;
    }

    let cancelled = false;
    setDetailLoading(true);
    api
      .getNodeDetail(selected.id, scopeId)
      .then((node) => {
        if (!cancelled) {
          setDetail(toNodeRows([node])[0] ?? null);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "failed to load server details");
          setDetail(null);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setDetailLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [api, scopeId, selected]);

  const onlineCount = useMemo(() => items.filter((item) => item.status === "active").length, [items]);
  const pendingCount = useMemo(() => items.filter((item) => item.status === "pending").length, [items]);
  const revokedCount = useMemo(() => items.filter((item) => item.status === "revoked").length, [items]);

  const onCreateServer = async () => {
    if (!scopeId || !canWrite) {
      return;
    }
    const hostname = createForm.hostname.trim();
    const region = createForm.region.trim();
    if (!hostname || !region) {
      setError("Hostname and region are required");
      return;
    }

    setCreatePending(true);
    try {
      await api.createNode({
        tenant_id: scopeId,
        hostname,
        region,
        agent_version: createForm.agentVersion.trim() || undefined,
        capabilities: createForm.capabilities,
      });
      setShowAddServer(false);
      setCreateForm(DEFAULT_FORM);
      await load();
      emitAdminDataChanged();
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "create server failed");
    } finally {
      setCreatePending(false);
    }
  };

  const setNodeStatus = async (row: NodeRowVM, target: "revoked" | "pending") => {
    if (!scopeId || !canWrite) {
      return;
    }
    const actionLabel = target === "revoked" ? "disable" : "reactivate";
    if (!window.confirm(`Do you want to ${actionLabel} ${row.hostname}?`)) {
      return;
    }

    setRowPendingId(row.id);
    try {
      if (target === "revoked") {
        await api.revokeNode({ tenant_id: scopeId, node_id: row.id });
      } else {
        await api.reactivateNode({ tenant_id: scopeId, node_id: row.id });
      }
      await load();
      emitAdminDataChanged();
      if (selected?.id === row.id) {
        const node = await api.getNodeDetail(row.id, scopeId);
        setDetail(toNodeRows([node])[0] ?? null);
      }
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : `failed to ${actionLabel} server`);
    } finally {
      setRowPendingId("");
    }
  };

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Servers"
          subtitle="Manage your VPN servers and locations"
          actions={
            <ActionButton variant="primary" onClick={() => setShowAddServer(true)} disabled={!canWrite || createPending}>
              <Plus size={15} /> Add Server
            </ActionButton>
          }
        />

        {error ? <ErrorState message={error} /> : null}

        <section className="nd-stat-grid">
          <StatCard label="Total Servers" value={String(items.length)} delta="0%" tone="neutral" />
          <StatCard label="Online" value={String(onlineCount)} delta="0%" tone="success" />
          <StatCard label="Pending" value={String(pendingCount)} delta="0%" tone="warning" />
          <StatCard label="Revoked" value={String(revokedCount)} delta="0%" tone={revokedCount > 0 ? "danger" : "neutral"} />
        </section>

        <div className="row" style={{ alignItems: "center" }}>
          <select value={status} onChange={(event) => setStatus(event.target.value)} style={{ maxWidth: 220 }}>
            <option value="">All status</option>
            <option value="active">Online</option>
            <option value="pending">Pending</option>
            <option value="stale">Stale</option>
            <option value="offline">Offline</option>
            <option value="revoked">Revoked</option>
          </select>
          <ActionButton variant="secondary" onClick={() => void load()} disabled={loading}>
            <RefreshCw size={15} /> {loading ? "Refreshing..." : "Refresh"}
          </ActionButton>
          <SupportBadge state="frontend_seam" />
        </div>

        {loading ? <LoadingState label="Loading servers..." /> : null}
        {!loading && items.length === 0 ? <EmptyState title="No servers yet" description="Add your first server to start accepting user traffic." /> : null}

        <section className="nd-server-grid">
          {items.map((row) => {
            const pending = rowPendingId === row.id;
            return (
              <article key={row.id} className="nd-server-card">
                <header className="nd-server-head">
                  <div className="nd-server-title">
                    <span className="nd-server-flag">{row.countryCode}</span>
                    <div>
                      <strong>{row.serverName}</strong>
                      <small>{row.hostname}</small>
                    </div>
                  </div>
                  <StatusBadge value={row.status === "active" ? "Online" : row.status} />
                </header>

                <div className="nd-server-metrics">
                  <span style={{ color: "var(--nd-text-secondary)" }}>Status telemetry is not exposed as load percentages yet.</span>
                </div>

                <div className="row" style={{ justifyContent: "space-between", margin: 0 }}>
                  <span style={{ color: "var(--nd-text-secondary)" }}>Region: {row.region}</span>
                  <span style={{ color: "var(--nd-text-secondary)" }}>Load: No data</span>
                </div>

                <div className="row" style={{ justifyContent: "space-between", margin: 0 }}>
                  <span style={{ color: "var(--nd-text-secondary)" }}>Last seen: {row.lastSeen}</span>
                  <span style={{ color: "var(--nd-text-secondary)" }}>Heartbeat: {row.heartbeat}</span>
                </div>

                <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                  {row.capabilities.length === 0 ? (
                    <ProtocolChip label="No capabilities" />
                  ) : (
                    row.capabilities.map((capability) => <ProtocolChip key={capability} label={capability} />)
                  )}
                </div>

                <footer className="nd-server-actions">
                  {row.status !== "revoked" ? (
                    <ActionButton
                      variant="danger"
                      onClick={() => void setNodeStatus(row, "revoked")}
                      disabled={!canWrite || pending}
                    >
                      {pending ? "Disabling..." : "Disable"}
                    </ActionButton>
                  ) : (
                    <ActionButton
                      variant="secondary"
                      onClick={() => void setNodeStatus(row, "pending")}
                      disabled={!canWrite || pending}
                    >
                      {pending ? "Reactivating..." : "Reactivate"}
                    </ActionButton>
                  )}

                  <div className="nd-inline-icons">
                    <button className="nd-icon-btn is-secondary" type="button" onClick={() => setSelected(row)} aria-label="Diagnostics">
                      <Activity size={15} />
                    </button>
                    <button className="nd-icon-btn is-secondary" type="button" disabled aria-label="Restart (frontend seam)">
                      <RotateCcw size={15} />
                    </button>
                    <button className="nd-icon-btn is-secondary" type="button" disabled aria-label="Edit (frontend seam)">
                      <Pencil size={15} />
                    </button>
                    <button className="nd-icon-btn is-danger" type="button" disabled aria-label="Delete (frontend seam)">
                      <Trash2 size={15} />
                    </button>
                  </div>
                </footer>
              </article>
            );
          })}
        </section>

        <DrawerShell
          title={selected ? `Server diagnostics · ${selected.hostname}` : "Server diagnostics"}
          open={Boolean(selected)}
          onClose={() => {
            setSelected(null);
            setDetail(null);
          }}
        >
          {detailLoading ? (
            <LoadingState label="Loading server diagnostics..." />
          ) : detail ? (
            <div style={{ display: "grid", gap: 12 }}>
              <p>
                <strong>{detail.serverName}</strong> · {detail.region}
              </p>
              <StatusBadge value={detail.status} />
              <p style={{ color: "var(--nd-text-secondary)" }}>Load telemetry: No data</p>
              <p>Last seen: {detail.lastSeen}</p>
              <p>Last heartbeat: {detail.heartbeat}</p>

              <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                {detail.capabilities.map((capability) => (
                  <ProtocolChip key={capability} label={capability} />
                ))}
              </div>

              <hr style={{ borderColor: "var(--nd-border-soft)", width: "100%" }} />

              <div style={{ display: "grid", gap: 6 }}>
                <p>
                  Node ID: <code>{detail.id}</code>
                </p>
                <p>
                  Node Key: <code>{detail.advanced.nodeKeyId}</code>
                </p>
                <p>
                  Agent: <code>{detail.advanced.agentVersion}</code>
                </p>
                <p>
                  Contract: <code>{detail.advanced.contractVersion}</code>
                </p>
                <p>
                  Identity Fingerprint: <code>{detail.advanced.identityFingerprint || "n/a"}</code>
                </p>
              </div>
            </div>
          ) : (
            <EmptyState title="No diagnostics yet" description="Select a server to see runtime details." />
          )}
        </DrawerShell>

        <ModalShell
          title="Add New Server"
          open={showAddServer}
          onClose={() => {
            setShowAddServer(false);
            setCreateForm(DEFAULT_FORM);
          }}
        >
          <div style={{ display: "grid", gap: 12 }}>
            <label>
              Hostname
              <input
                value={createForm.hostname}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, hostname: event.target.value }))}
                placeholder="de-1.example.com"
              />
            </label>

            <label>
              Region
              <input
                value={createForm.region}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, region: event.target.value }))}
                placeholder="de-central"
              />
            </label>

            <label>
              Agent Version
              <input
                value={createForm.agentVersion}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, agentVersion: event.target.value }))}
                placeholder="owner-manual"
              />
            </label>

            <div>
              <div style={{ marginBottom: 6, fontWeight: 600 }}>Capabilities</div>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                {CAPABILITY_OPTIONS.map((capability) => {
                  const checked = createForm.capabilities.includes(capability);
                  return (
                    <label key={capability} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={(event) => {
                          setCreateForm((prev) => ({
                            ...prev,
                            capabilities: event.target.checked
                              ? [...prev.capabilities, capability]
                              : prev.capabilities.filter((item) => item !== capability),
                          }));
                        }}
                        style={{ width: 16, height: 16 }}
                      />
                      {capability}
                    </label>
                  );
                })}
              </div>
            </div>

            <div style={{ marginTop: 6, display: "flex", justifyContent: "flex-end", gap: 8 }}>
              <ActionButton
                variant="ghost"
                onClick={() => {
                  setShowAddServer(false);
                  setCreateForm(DEFAULT_FORM);
                }}
                disabled={createPending}
              >
                Cancel
              </ActionButton>
              <ActionButton variant="primary" onClick={() => void onCreateServer()} disabled={!canWrite || createPending}>
                <Zap size={14} /> {createPending ? "Adding..." : "Add Server"}
              </ActionButton>
            </div>
          </div>
        </ModalShell>
      </AdminShell>
    </RouteGuard>
  );
}


