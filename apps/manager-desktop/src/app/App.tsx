import { useEffect, useMemo, useState } from "react";
import {
  OpenerNetDoorClient,
  createLocalPlannerServerOperationsAdapter,
  type ServerOperationsAdapter,
} from "@opener-netdoor/sdk-ts";
import {
  PROTOCOL_CATALOG,
  type AccessKey,
  type EffectivePolicy,
  type Node,
  type OpsSnapshot,
  type ServerInventoryItem,
  type ServerOperationJob,
  type ServerOperationPlan,
  type ServerOperationType,
  type TenantPolicy,
} from "@opener-netdoor/shared-types";
import { diagnostics } from "../platform/ipc";
import { getSecret, setSecret } from "../platform/secure_storage";
import { checkForUpdates } from "../platform/updater";
import { clearSession, getSession, hasScope, saveSession, type ManagerSession } from "../state/session";

type Route =
  | "login"
  | "dashboard"
  | "nodes"
  | "keys"
  | "policies"
  | "servers"
  | "protocols"
  | "diagnostics";

const NAV_ITEMS: Array<{ route: Route; label: string; requiredScope: string }> = [
  { route: "dashboard", label: "Dashboard", requiredScope: "admin:read" },
  { route: "nodes", label: "Nodes", requiredScope: "admin:read" },
  { route: "keys", label: "Access Keys", requiredScope: "admin:read" },
  { route: "policies", label: "Policies", requiredScope: "admin:read" },
  { route: "servers", label: "Server Ops", requiredScope: "admin:read" },
  { route: "protocols", label: "Protocols", requiredScope: "admin:read" },
  { route: "diagnostics", label: "Diagnostics", requiredScope: "admin:read" },
];

const SERVER_OPERATIONS: ServerOperationType[] = [
  "install",
  "upgrade",
  "rollback",
  "backup",
  "restore",
  "cert_rotate",
  "firewall_apply",
  "cdn_apply",
];

export function App() {
  const [session, setSessionState] = useState<ManagerSession | null>(null);
  const [route, setRoute] = useState<Route>("login");
  const [error, setError] = useState("");
  const [diagnosticData, setDiagnosticData] = useState<Record<string, unknown> | null>(null);
  const [theme, setTheme] = useState<"dark" | "light">("dark");

  useEffect(() => {
    const existing = getSession();
    setSessionState(existing);
    if (existing) {
      setTheme(existing.theme);
      setRoute("dashboard");
    }
    void checkForUpdates();
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  const client = useMemo(() => {
    if (!session) {
      return null;
    }
    return new OpenerNetDoorClient({
      baseUrl: session.baseUrl,
      token: session.token,
      tenantId: session.tenantId,
    });
  }, [session]);

  const logout = () => {
    clearSession();
    setSessionState(null);
    setDiagnosticData(null);
    setError("");
    setRoute("login");
  };

  if (!session || route === "login") {
    return (
      <LoginScreen
        onLogin={(next) => {
          setSessionState(next);
          setTheme(next.theme);
          setRoute("dashboard");
          setError("");
        }}
      />
    );
  }

  return (
    <main className="desktop-shell">
      <aside className="desktop-sidebar">
        <h1>Opener NetDoor Manager</h1>
        <p className="subtitle">Operator workstation</p>
        <p>
          actor <code>{session.subject ?? "unknown"}</code>
        </p>
        <p>
          tenant <code>{session.tenantId ?? "platform"}</code>
        </p>
        <div className="desktop-nav">
          {NAV_ITEMS.filter((item) => hasScope(session, item.requiredScope)).map((item) => (
            <button key={item.route} className={route === item.route ? "active" : ""} onClick={() => setRoute(item.route)}>
              {item.label}
            </button>
          ))}
          <button className="secondary" onClick={() => setTheme(theme === "dark" ? "light" : "dark")}> 
            Theme: {theme}
          </button>
          <button className="danger" onClick={logout}>
            Logout
          </button>
        </div>
      </aside>
      <section className="desktop-content">
        {error ? <p className="error">{error}</p> : null}
        {route === "dashboard" && <DashboardScreen client={client} />}
        {route === "nodes" && (
          <NodesScreen
            client={client}
            tenantId={session.tenantId}
            canWrite={hasScope(session, "admin:write")}
            onError={setError}
          />
        )}
        {route === "keys" && (
          <KeysScreen
            client={client}
            tenantId={session.tenantId}
            canWrite={hasScope(session, "admin:write")}
            onError={setError}
          />
        )}
        {route === "policies" && <PoliciesScreen client={client} tenantId={session.tenantId} onError={setError} />}
        {route === "servers" && <ServersScreen onError={setError} />}
        {route === "protocols" && <ProtocolsScreen />}
        {route === "diagnostics" && (
          <DiagnosticsScreen
            data={diagnosticData}
            onRun={async () => {
              try {
                setError("");
                const secret = await getSecret("manager:last-token");
                if (!secret && session.token) {
                  await setSecret("manager:last-token", session.token);
                }
                setDiagnosticData(await diagnostics());
              } catch (err) {
                setError(err instanceof Error ? err.message : "diagnostics failed");
              }
            }}
          />
        )}
      </section>
    </main>
  );
}

function LoginScreen({ onLogin }: { onLogin: (session: ManagerSession) => void }) {
  const [baseUrl, setBaseUrl] = useState("http://127.0.0.1:8080");
  const [token, setToken] = useState("");
  const [tenantId, setTenantId] = useState("");
  const [theme, setTheme] = useState<"dark" | "light">("dark");

  return (
    <main className="login-root">
      <div className="panel">
        <h2>Manager Login</h2>
        <p>JWT token is required. Scope parsing is extracted from token payload.</p>
        <div className="form-grid">
          <label>
            API URL
            <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} />
          </label>
          <label>
            Tenant ID (optional)
            <input value={tenantId} onChange={(event) => setTenantId(event.target.value)} />
          </label>
          <label>
            Theme
            <select value={theme} onChange={(event) => setTheme(event.target.value as "dark" | "light")}> 
              <option value="dark">dark</option>
              <option value="light">light</option>
            </select>
          </label>
          <label>
            Token
            <textarea value={token} rows={5} onChange={(event) => setToken(event.target.value)} />
          </label>
          <button
            onClick={() => {
              const saved = saveSession({
                baseUrl,
                token,
                tenantId: tenantId.trim() || undefined,
                theme,
              });
              onLogin(saved);
            }}
          >
            Open manager
          </button>
        </div>
      </div>
    </main>
  );
}

function DashboardScreen({ client }: { client: OpenerNetDoorClient | null }) {
  const [health, setHealth] = useState("loading");
  const [ready, setReady] = useState("loading");
  const [snapshot, setSnapshot] = useState<OpsSnapshot | null>(null);

  useEffect(() => {
    if (!client) {
      return;
    }
    void client.health().then((result) => setHealth(result.status)).catch(() => setHealth("error"));
    void client.ready().then((result) => setReady(result.status)).catch(() => setReady("error"));
    void client.opsSnapshot().then(setSnapshot).catch(() => setSnapshot(null));
  }, [client]);

  return (
    <section className="panel">
      <h2>Dashboard</h2>
      <div className="metrics-grid">
        <MetricCard title="Health" value={health} />
        <MetricCard title="Ready" value={ready} />
        <MetricCard title="Replay rejects 24h" value={snapshot ? String(snapshot.replay_rejected_24h) : "n/a"} />
        <MetricCard title="Invalid signatures 24h" value={snapshot ? String(snapshot.invalid_signature_24h) : "n/a"} />
      </div>
      {snapshot ? <pre>{JSON.stringify(snapshot, null, 2)}</pre> : null}
    </section>
  );
}

function NodesScreen({
  client,
  tenantId,
  canWrite,
  onError,
}: {
  client: OpenerNetDoorClient | null;
  tenantId?: string;
  canWrite: boolean;
  onError: (value: string) => void;
}) {
  const [items, setItems] = useState<Node[]>([]);

  const load = () => {
    if (!client) {
      return;
    }
    if (!tenantId) {
      onError("tenant_id is required for node operations in tenant-scoped manager mode");
      return;
    }
    void client
      .listNodesPage({ tenantId, limit: 30, offset: 0 })
      .then((result) => {
        setItems(result.items);
        onError("");
      })
      .catch((err) => onError(err instanceof Error ? err.message : "nodes request failed"));
  };

  useEffect(() => {
    load();
  }, [client, tenantId]);

  return (
    <section className="panel">
      <div className="panel-head">
        <h2>Nodes</h2>
        <button onClick={load}>Refresh</button>
      </div>
      <table className="desktop-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Host</th>
            <th>Region</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((node) => (
            <tr key={node.id}>
              <td>
                <code>{node.id}</code>
              </td>
              <td>{node.hostname}</td>
              <td>{node.region}</td>
              <td>{node.status}</td>
              <td>
                {!canWrite || !tenantId ? (
                  "read-only"
                ) : node.status === "revoked" ? (
                  <button
                    onClick={() => {
                      void client
                        ?.reactivateNode({ tenant_id: tenantId, node_id: node.id })
                        .then(load)
                        .catch((err) => onError(err instanceof Error ? err.message : "reactivate failed"));
                    }}
                  >
                    Reactivate
                  </button>
                ) : (
                  <button
                    className="danger"
                    onClick={() => {
                      void client
                        ?.revokeNode({ tenant_id: tenantId, node_id: node.id })
                        .then(load)
                        .catch((err) => onError(err instanceof Error ? err.message : "revoke failed"));
                    }}
                  >
                    Revoke
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function KeysScreen({
  client,
  tenantId,
  canWrite,
  onError,
}: {
  client: OpenerNetDoorClient | null;
  tenantId?: string;
  canWrite: boolean;
  onError: (value: string) => void;
}) {
  const [items, setItems] = useState<AccessKey[]>([]);

  const load = () => {
    if (!client) {
      return;
    }
    void client
      .listAccessKeysPage({ tenantId, limit: 30, offset: 0 })
      .then((result) => {
        setItems(result.items);
        onError("");
      })
      .catch((err) => onError(err instanceof Error ? err.message : "keys request failed"));
  };

  useEffect(() => {
    load();
  }, [client, tenantId]);

  return (
    <section className="panel">
      <div className="panel-head">
        <h2>Access Keys</h2>
        <button onClick={load}>Refresh</button>
      </div>
      <table className="desktop-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>User</th>
            <th>Status</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {items.map((key) => (
            <tr key={key.id}>
              <td>
                <code>{key.id}</code>
              </td>
              <td>
                <code>{key.user_id}</code>
              </td>
              <td>{key.status}</td>
              <td>
                {!canWrite || key.status === "revoked" ? (
                  "read-only"
                ) : (
                  <button
                    className="danger"
                    onClick={() => {
                      void client
                        ?.revokeAccessKey(key.id, key.tenant_id)
                        .then(load)
                        .catch((err) => onError(err instanceof Error ? err.message : "revoke failed"));
                    }}
                  >
                    Revoke
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function PoliciesScreen({
  client,
  tenantId,
  onError,
}: {
  client: OpenerNetDoorClient | null;
  tenantId?: string;
  onError: (value: string) => void;
}) {
  const [policies, setPolicies] = useState<TenantPolicy[]>([]);
  const [effective, setEffective] = useState<EffectivePolicy | null>(null);
  const [userId, setUserId] = useState("");

  useEffect(() => {
    if (!client || !tenantId) {
      return;
    }
    void client
      .listTenantPoliciesPage({ tenantId, limit: 20, offset: 0 })
      .then((result) => {
        setPolicies(result.items);
        onError("");
      })
      .catch((err) => onError(err instanceof Error ? err.message : "policies request failed"));
  }, [client, tenantId]);

  return (
    <section className="panel">
      <h2>Policies</h2>
      <p>Tenant defaults and user effective policy lookup.</p>
      <pre>{JSON.stringify(policies, null, 2)}</pre>

      <div className="form-grid">
        <label>
          user_id for effective lookup
          <input value={userId} onChange={(event) => setUserId(event.target.value)} />
        </label>
        <button
          onClick={() => {
            if (!client || !tenantId || !userId.trim()) {
              onError("tenant_id and user_id are required for effective policy lookup");
              return;
            }
            void client
              .getEffectivePolicy(tenantId, userId.trim())
              .then((result) => {
                setEffective(result);
                onError("");
              })
              .catch((err) => onError(err instanceof Error ? err.message : "effective policy lookup failed"));
          }}
        >
          Resolve
        </button>
      </div>

      {effective ? <pre>{JSON.stringify(effective, null, 2)}</pre> : null}
    </section>
  );
}

function ServersScreen({ onError }: { onError: (value: string) => void }) {
  const adapter: ServerOperationsAdapter = useMemo(() => createLocalPlannerServerOperationsAdapter(), []);
  const [servers, setServers] = useState<ServerInventoryItem[]>([]);
  const [jobs, setJobs] = useState<ServerOperationJob[]>([]);
  const [selectedServer, setSelectedServer] = useState("");
  const [operation, setOperation] = useState<ServerOperationType>("install");
  const [plan, setPlan] = useState<ServerOperationPlan | null>(null);

  useEffect(() => {
    void adapter
      .listServers()
      .then((items) => {
        setServers(items);
        if (items[0]) {
          setSelectedServer(items[0].id);
        }
      })
      .catch((err) => onError(err instanceof Error ? err.message : "server list failed"));

    void adapter
      .listJobs()
      .then(setJobs)
      .catch((err) => onError(err instanceof Error ? err.message : "job list failed"));
  }, [adapter]);

  return (
    <section className="panel">
      <h2>Server Operations</h2>
      <p>Local planner seam for install/upgrade/rollback/backup actions while backend execution API is staged.</p>
      <div className="form-grid">
        <label>
          server
          <select value={selectedServer} onChange={(event) => setSelectedServer(event.target.value)}>
            {servers.map((server) => (
              <option key={server.id} value={server.id}>
                {server.id} ({server.region})
              </option>
            ))}
          </select>
        </label>
        <label>
          operation
          <select value={operation} onChange={(event) => setOperation(event.target.value as ServerOperationType)}>
            {SERVER_OPERATIONS.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
        </label>
        <div className="button-row">
          <button
            onClick={() => {
              if (!selectedServer) {
                onError("select server first");
                return;
              }
              void adapter
                .createPlan({ serverId: selectedServer, operation })
                .then((result) => {
                  setPlan(result);
                  onError("");
                })
                .catch((err) => onError(err instanceof Error ? err.message : "plan failed"));
            }}
          >
            Preview
          </button>
          <button
            className="secondary"
            onClick={() => {
              if (!selectedServer) {
                onError("select server first");
                return;
              }
              void adapter
                .startOperation({ serverId: selectedServer, operation })
                .then(() => adapter.listJobs())
                .then((result) => {
                  setJobs(result);
                  onError("");
                })
                .catch((err) => onError(err instanceof Error ? err.message : "operation failed"));
            }}
          >
            Run
          </button>
        </div>
      </div>

      {plan ? <pre>{JSON.stringify(plan, null, 2)}</pre> : null}
      <pre>{JSON.stringify(jobs, null, 2)}</pre>
    </section>
  );
}

function ProtocolsScreen() {
  return (
    <section className="panel">
      <h2>Protocol Catalog</h2>
      <p>`Nieva` remains unverified and intentionally unsupported.</p>
      <table className="desktop-table">
        <thead>
          <tr>
            <th>Protocol</th>
            <th>Family</th>
            <th>CDN</th>
            <th>QUIC</th>
            <th>Support</th>
          </tr>
        </thead>
        <tbody>
          {PROTOCOL_CATALOG.map((entry) => (
            <tr key={entry.id}>
              <td>{entry.label}</td>
              <td>{entry.family}</td>
              <td>{entry.supports_cdn ? "yes" : "no"}</td>
              <td>{entry.supports_quic ? "yes" : "no"}</td>
              <td>{entry.support}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function DiagnosticsScreen({
  data,
  onRun,
}: {
  data: Record<string, unknown> | null;
  onRun: () => Promise<void>;
}) {
  return (
    <section className="panel">
      <h2>Diagnostics</h2>
      <div className="button-row">
        <button onClick={() => void onRun()}>Run diagnostics</button>
        <button
          className="secondary"
          onClick={() => {
            const payload = JSON.stringify(data ?? { status: "not started" }, null, 2);
            const blob = new Blob([payload], { type: "application/json" });
            const url = URL.createObjectURL(blob);
            const anchor = document.createElement("a");
            anchor.href = url;
            anchor.download = `opener-netdoor-diagnostics-${Date.now()}.json`;
            anchor.click();
            URL.revokeObjectURL(url);
          }}
        >
          Export JSON
        </button>
      </div>
      <pre>{JSON.stringify(data ?? { status: "not started" }, null, 2)}</pre>
    </section>
  );
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <div className="metric-card">
      <span>{title}</span>
      <strong>{value}</strong>
    </div>
  );
}
