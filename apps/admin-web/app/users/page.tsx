"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Ban, Copy, KeyRound, Pencil, Plus, Trash2, Unlock } from "lucide-react";
import type { AccessKey } from "@opener-netdoor/shared-types";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import {
  ActionButton,
  Card,
  DataTable,
  ErrorState,
  IconButton,
  ModalShell,
  PageTitle,
  PaginationControls,
  ProgressBar,
  SearchInput,
  StatusBadge,
} from "../../components/ui";
import { useAPIClient, useAdminSession, useOwnerScopeId } from "../../lib/api/client";
import { toUserRows, type UserRowVM } from "../../lib/adapters/users";
import { formatBytes, formatDateTime } from "../../lib/format";
import { emitAdminDataChanged } from "../../lib/events/admin-data";
import { hasScopes } from "../../lib/permissions";

interface CreateUserForm {
  name: string;
  email: string;
  subscription: "Basic" | "Standard" | "Premium" | "Enterprise";
  trafficLimitGB: number;
}

const DEFAULT_FORM: CreateUserForm = {
  name: "",
  email: "",
  subscription: "Basic",
  trafficLimitGB: 100,
};

const KEY_TYPES = ["vless", "vmess", "trojan", "shadowsocks"] as const;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function UsersPage() {
  const api = useAPIClient();
  const session = useAdminSession();
  const scopeId = useOwnerScopeId();

  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);

  const [items, setItems] = useState<UserRowVM[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [error, setError] = useState("");

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<CreateUserForm>(DEFAULT_FORM);
  const [createPending, setCreatePending] = useState(false);

  const [selectedUser, setSelectedUser] = useState<UserRowVM | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<AccessKey[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [keysError, setKeysError] = useState("");
  const [keyCreatingType, setKeyCreatingType] = useState<string>("");
  const [keyRevokingID, setKeyRevokingID] = useState("");
  const [createdKeyMaterial, setCreatedKeyMaterial] = useState<AccessKey | null>(null);
  const [copiedKeyId, setCopiedKeyId] = useState("");

  const [mutatingUserId, setMutatingUserId] = useState("");

  const canWrite = hasScopes(session?.scopes ?? [], ["admin:write"]);

  const loadUsers = useCallback(async () => {
    if (!scopeId) {
      setError("Hidden owner scope is not available in session");
      return;
    }
    setLoadingUsers(true);
    try {
      const usersRes = await api.listUsersPage({ tenantId: scopeId, status: status || undefined, limit, offset });
      setItems(toUserRows(usersRes.items));
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "request failed");
    } finally {
      setLoadingUsers(false);
    }
  }, [api, limit, offset, scopeId, status]);

  const loadKeysForUser = useCallback(
    async (userId: string) => {
      if (!scopeId) {
        setKeysError("Hidden owner scope is not available in session");
        return;
      }
      setKeysLoading(true);
      try {
        const result = await api.listAccessKeysPage({ tenantId: scopeId, userId, limit: 200, offset: 0 });
        setSelectedKeys(result.items);
        setKeysError("");
      } catch (err) {
        setSelectedKeys([]);
        setKeysError(err instanceof Error ? err.message : "failed to load access keys");
      } finally {
        setKeysLoading(false);
      }
    },
    [api, scopeId],
  );

  useEffect(() => {
    if (scopeId) {
      void loadUsers();
    }
  }, [loadUsers, scopeId]);

  useEffect(() => {
    if (!selectedUser) {
      return;
    }
    setSelectedKeys([]);
    setCreatedKeyMaterial(null);
    setCopiedKeyId("");
    void loadKeysForUser(selectedUser.id);
  }, [loadKeysForUser, selectedUser]);

  const filteredRows = useMemo(() => {
    const query = search.trim().toLowerCase();
    return items.filter((item) => {
      if (status && item.status.toLowerCase() !== status.toLowerCase()) {
        return false;
      }
      if (!query) {
        return true;
      }
      return item.name.toLowerCase().includes(query) || item.email.toLowerCase().includes(query);
    });
  }, [items, search, status]);

  const resetCreateForm = () => {
    setCreateForm(DEFAULT_FORM);
    setCreatePending(false);
  };

  const onCreateUser = async () => {
    if (!scopeId) {
      setError("Hidden owner scope is not available in session");
      return;
    }

    const email = createForm.email.trim();
    if (!email) {
      setError("Email is required");
      return;
    }
    if (!EMAIL_RE.test(email)) {
      setError("Email format is invalid");
      return;
    }

    setCreatePending(true);
    const note = JSON.stringify({
      display_name: createForm.name.trim() || undefined,
      subscription: createForm.subscription,
      traffic_limit_gb: createForm.trafficLimitGB,
      source: "admin_web_form",
      mode: "single_owner",
    });

    try {
      await api.createUser({
        tenant_id: scopeId,
        email,
        note,
      });
      setShowCreate(false);
      resetCreateForm();
      await loadUsers();
      emitAdminDataChanged();
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "create user failed");
    } finally {
      setCreatePending(false);
    }
  };

  const mutateUserStatus = async (row: UserRowVM, target: "blocked" | "active") => {
    if (!scopeId) {
      setError("Hidden owner scope is not available in session");
      return;
    }
    if (!canWrite) {
      return;
    }

    const actionLabel = target === "blocked" ? "block" : "unblock";
    if (!window.confirm(`Do you want to ${actionLabel} ${row.email}?`)) {
      return;
    }

    setMutatingUserId(row.id);
    try {
      if (target === "blocked") {
        await api.blockUser({ tenant_id: scopeId, user_id: row.id });
      } else {
        await api.unblockUser({ tenant_id: scopeId, user_id: row.id });
      }
      await loadUsers();
      emitAdminDataChanged();
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : `failed to ${actionLabel} user`);
    } finally {
      setMutatingUserId("");
    }
  };

  const onDeleteUser = async (row: UserRowVM) => {
    if (!scopeId || !canWrite) {
      return;
    }
    if (!window.confirm(`Delete ${row.email}? This removes all user keys as well.`)) {
      return;
    }

    setMutatingUserId(row.id);
    try {
      await api.deleteUser(row.id, scopeId);
      if (selectedUser?.id === row.id) {
        setSelectedUser(null);
        setSelectedKeys([]);
        setCreatedKeyMaterial(null);
      }
      await loadUsers();
      emitAdminDataChanged();
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "delete user failed");
    } finally {
      setMutatingUserId("");
    }
  };

  const onCreateKey = async (keyType: string) => {
    if (!scopeId || !selectedUser || !canWrite) {
      return;
    }
    setKeyCreatingType(keyType);
    try {
      const created = await api.createAccessKey({
        tenant_id: scopeId,
        user_id: selectedUser.id,
        key_type: keyType,
      });
      setCreatedKeyMaterial(created);
      await loadKeysForUser(selectedUser.id);
      emitAdminDataChanged();
      setKeysError("");
    } catch (err) {
      setKeysError(err instanceof Error ? err.message : "create key failed");
    } finally {
      setKeyCreatingType("");
    }
  };

  const onRevokeKey = async (key: AccessKey) => {
    if (!canWrite) {
      return;
    }
    if (!window.confirm(`Revoke key ${key.id}?`)) {
      return;
    }
    setKeyRevokingID(key.id);
    try {
      await api.revokeAccessKey(key.id, key.tenant_id);
      if (selectedUser) {
        await loadKeysForUser(selectedUser.id);
      }
      emitAdminDataChanged();
      setKeysError("");
    } catch (err) {
      setKeysError(err instanceof Error ? err.message : "revoke key failed");
    } finally {
      setKeyRevokingID("");
    }
  };

  const keyMaterial = (key: AccessKey): string => key.connection_uri?.trim() || key.secret_ref;

  const onCopy = async (keyId: string, value: string) => {
    if (!value) {
      return;
    }
    try {
      await window.navigator.clipboard.writeText(value);
      setCopiedKeyId(keyId);
      window.setTimeout(() => setCopiedKeyId(""), 1200);
    } catch {
      setKeysError("copy failed");
    }
  };

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle
          title="Users"
          subtitle="Manage user accounts and access keys"
          actions={
            <ActionButton variant="primary" onClick={() => setShowCreate(true)} disabled={!canWrite || createPending}>
              <Plus size={15} /> Add User
            </ActionButton>
          }
        />

        {error ? <ErrorState message={error} /> : null}

        <Card>
          <div className="row" style={{ width: "100%", alignItems: "center" }}>
            <div style={{ flex: 1, minWidth: 280 }}>
              <SearchInput value={search} onChange={setSearch} placeholder="Search users..." />
            </div>
            <select style={{ maxWidth: 180 }} value={status} onChange={(event) => setStatus(event.target.value)}>
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="blocked">Blocked</option>
              <option value="expired">Expired</option>
            </select>
            <ActionButton variant="secondary" onClick={() => void loadUsers()} disabled={loadingUsers}>
              {loadingUsers ? "Refreshing..." : "Refresh"}
            </ActionButton>
          </div>
        </Card>

        <Card className="nd-activity-table">
          <DataTable
            rows={filteredRows}
            rowKey={(row) => row.id}
            columns={[
              {
                id: "user",
                header: "User",
                render: (row) => (
                  <div className="nd-user-cell">
                    <span className="nd-avatar">{row.name[0]?.toUpperCase() ?? "U"}</span>
                    <div className="nd-user-meta">
                      <strong>{row.name}</strong>
                      <small>{row.email}</small>
                    </div>
                  </div>
                ),
              },
              { id: "status", header: "Status", render: (row) => <StatusBadge value={row.status} /> },
              {
                id: "traffic",
                header: "Traffic",
                render: (row) => (
                  <div style={{ minWidth: 180 }}>
                    {typeof row.trafficPercent === "number" &&
                    typeof row.trafficUsedBytes === "number" &&
                    typeof row.trafficLimitBytes === "number" ? (
                      <ProgressBar
                        value={row.trafficPercent}
                        hint={`${formatBytes(row.trafficUsedBytes)} / ${formatBytes(row.trafficLimitBytes)}`}
                      />
                    ) : (
                      <span style={{ color: "var(--nd-text-muted)" }}>No data</span>
                    )}
                  </div>
                ),
              },
              {
                id: "subscription",
                header: "Subscription",
                render: (row) => (
                  <span
                    style={{
                      color:
                        row.subscription === "Enterprise"
                          ? "var(--nd-chart-orange)"
                          : row.subscription === "Premium"
                            ? "var(--nd-chart-yellow)"
                            : row.subscription === "Standard"
                              ? "var(--nd-success)"
                              : "var(--nd-text-secondary)",
                      fontWeight: 600,
                    }}
                  >
                    {row.subscription ?? "No data"}
                  </span>
                ),
              },
              { id: "expiry", header: "Expiry", render: (row) => (row.expiresAt ? formatDateTime(row.expiresAt) : <span style={{ color: "var(--nd-text-muted)" }}>No data</span>) },
              {
                id: "actions",
                header: "Actions",
                render: (row) => {
                  const mutationPending = mutatingUserId === row.id;
                  const isBlocked = row.status === "Blocked";
                  return (
                    <div className="nd-row-actions">
                      <IconButton icon={<KeyRound size={15} />} label="Access keys" variant="secondary" onClick={() => setSelectedUser(row)} />
                      <IconButton icon={<Pencil size={15} />} label="Edit user" variant="secondary" disabled />
                      {isBlocked ? (
                        <IconButton
                          icon={<Unlock size={15} />}
                          label="Unblock user"
                          variant="secondary"
                          onClick={() => void mutateUserStatus(row, "active")}
                          disabled={!canWrite || mutationPending}
                        />
                      ) : (
                        <IconButton
                          icon={<Ban size={15} />}
                          label="Block user"
                          variant="secondary"
                          onClick={() => void mutateUserStatus(row, "blocked")}
                          disabled={!canWrite || mutationPending}
                        />
                      )}
                      <IconButton
                        icon={<Trash2 size={15} />}
                        label="Delete user"
                        variant="danger"
                        onClick={() => void onDeleteUser(row)}
                        disabled={!canWrite || mutationPending}
                      />
                    </div>
                  );
                },
              },
            ]}
          />

          <PaginationControls
            limit={limit}
            offset={offset}
            onChange={({ limit: nextLimit, offset: nextOffset }) => {
              setLimit(nextLimit);
              setOffset(nextOffset);
            }}
          />
        </Card>

        <ModalShell
          title="Add New User"
          open={showCreate}
          onClose={() => {
            setShowCreate(false);
            resetCreateForm();
          }}
        >
          <div style={{ display: "grid", gap: 12 }}>
            <label>
              Name
              <input
                value={createForm.name}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, name: event.target.value }))}
                placeholder="John Doe"
              />
            </label>
            <label>
              Email
              <input
                value={createForm.email}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, email: event.target.value }))}
                placeholder="user@example.com"
              />
            </label>
            <label>
              Subscription
              <select
                value={createForm.subscription}
                onChange={(event) =>
                  setCreateForm((prev) => ({
                    ...prev,
                    subscription: event.target.value as CreateUserForm["subscription"],
                  }))
                }
              >
                <option value="Basic">Basic</option>
                <option value="Standard">Standard</option>
                <option value="Premium">Premium</option>
                <option value="Enterprise">Enterprise</option>
              </select>
            </label>
            <label>
              Traffic Limit (GB)
              <input
                type="number"
                min={1}
                value={createForm.trafficLimitGB}
                onChange={(event) => setCreateForm((prev) => ({ ...prev, trafficLimitGB: Number(event.target.value || 0) }))}
              />
            </label>

            <div style={{ display: "flex", justifyContent: "flex-end", gap: 8, marginTop: 4 }}>
              <ActionButton
                variant="ghost"
                onClick={() => {
                  setShowCreate(false);
                  resetCreateForm();
                }}
                disabled={createPending}
              >
                Cancel
              </ActionButton>
              <ActionButton variant="primary" onClick={() => void onCreateUser()} disabled={!canWrite || createPending}>
                {createPending ? "Creating..." : "Create User"}
              </ActionButton>
            </div>
          </div>
        </ModalShell>

        <ModalShell
          title={selectedUser ? `Access Keys - ${selectedUser.name}` : "Access Keys"}
          open={Boolean(selectedUser)}
          onClose={() => {
            setSelectedUser(null);
            setSelectedKeys([]);
            setCreatedKeyMaterial(null);
            setCopiedKeyId("");
            setKeysError("");
          }}
        >
          {selectedUser ? (
            <div style={{ display: "grid", gap: 14 }}>
              {keysError ? <ErrorState message={keysError} /> : null}

              <section className="nd-card" style={{ padding: 14 }}>
                <div style={{ marginBottom: 8, fontWeight: 600 }}>Generate New Key</div>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                  {KEY_TYPES.map((protocol) => (
                    <button
                      key={protocol}
                      className="nd-btn is-secondary"
                      type="button"
                      disabled={!canWrite || keyCreatingType.length > 0}
                      onClick={() => void onCreateKey(protocol)}
                    >
                      {keyCreatingType === protocol ? "Creating..." : protocol.toUpperCase()}
                    </button>
                  ))}
                </div>
              </section>

              {createdKeyMaterial ? (
                <section className="nd-card" style={{ padding: 12 }}>
                  <div style={{ marginBottom: 8, fontWeight: 600 }}>Newly Created Key Material</div>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 8 }}>
                    <strong>{createdKeyMaterial.key_type.toUpperCase()} Key</strong>
                    <button
                      className="nd-icon-btn is-secondary"
                      type="button"
                      onClick={() => void onCopy(`new-${createdKeyMaterial.id}`, keyMaterial(createdKeyMaterial))}
                      aria-label="Copy newly created key material"
                    >
                      <Copy size={15} />
                    </button>
                  </div>
                  <code style={{ display: "block", marginTop: 8, overflowWrap: "anywhere" }}>{keyMaterial(createdKeyMaterial)}</code>
                  {copiedKeyId === `new-${createdKeyMaterial.id}` ? (
                    <span style={{ marginTop: 6, display: "inline-block", color: "var(--nd-success)", fontSize: 12 }}>Copied</span>
                  ) : null}
                </section>
              ) : null}

              <section>
                <div style={{ marginBottom: 8, fontWeight: 600 }}>Existing Keys</div>
                <div style={{ display: "grid", gap: 8 }}>
                  {keysLoading ? (
                    <p style={{ color: "var(--nd-text-muted)", margin: 0 }}>Loading keys...</p>
                  ) : selectedKeys.length === 0 ? (
                    <p style={{ color: "var(--nd-text-muted)", margin: 0 }}>No keys yet for this user.</p>
                  ) : (
                    selectedKeys.map((key) => (
                      <article key={key.id} className="nd-card" style={{ padding: 12 }}>
                        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 8 }}>
                          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                            <strong>{key.key_type.toUpperCase()} Key</strong>
                            <StatusBadge value={key.status} />
                          </div>
                          <div className="nd-row-actions">
                            <button
                              className="nd-icon-btn is-secondary"
                              type="button"
                              onClick={() => void onCopy(key.id, keyMaterial(key))}
                              aria-label="Copy key material"
                            >
                              <Copy size={15} />
                            </button>
                            <button
                              className="nd-icon-btn is-danger"
                              type="button"
                              onClick={() => void onRevokeKey(key)}
                              disabled={!canWrite || key.status === "revoked" || keyRevokingID === key.id}
                              aria-label="Revoke key"
                            >
                              <Ban size={15} />
                            </button>
                          </div>
                        </div>
                        <code style={{ display: "block", marginTop: 8, overflowWrap: "anywhere" }}>{keyMaterial(key)}</code>
                        <small style={{ color: "var(--nd-text-muted)", display: "block", marginTop: 6 }}>
                          Created {formatDateTime(key.created_at)}
                        </small>
                        {copiedKeyId === key.id ? (
                          <span style={{ marginTop: 6, display: "inline-block", color: "var(--nd-success)", fontSize: 12 }}>Copied</span>
                        ) : null}
                      </article>
                    ))
                  )}
                </div>
              </section>
            </div>
          ) : null}
        </ModalShell>
      </AdminShell>
    </RouteGuard>
  );
}





