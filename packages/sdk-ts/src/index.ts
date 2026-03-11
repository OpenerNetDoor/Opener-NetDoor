import type {
  AccessKey,
  AccessKeyListResponse,
  ActivatePKIIssuerRequest,
  AuditLogListResponse,
  CARotationResult,
  CreatePKIIssuerRequest,
  CreateTenantRequest,
  CreateUserRequest,
  CreateNodeRequest,
  CreateAccessKeyRequest,
  Device,
  EffectivePolicy,
  HealthResponse,
  Node,
  NodeCertificate,
  NodeCertificateListResponse,
  NodeLifecycleRequest,
  NodeListResponse,
  NodeProvisioningContract,
  OpsSnapshot,
  PaginationQuery,
  PKIIssuer,
  PKIIssuerListResponse,
  RegisterDeviceRequest,
  RenewNodeCertificateRequest,
  RenewNodeCertificateResult,
  RevokeNodeCertificateRequest,
  RotateNodeCertificateRequest,
  SetTenantPolicyRequest,
  SetUserPolicyOverrideRequest,
  Tenant,
  TenantListResponse,
  TenantPolicy,
  TenantPolicyListResponse,
  User,
  UserLifecycleRequest,
  UserListResponse,
  UserPolicyOverride,
  UserPolicyOverrideListResponse,
} from "@opener-netdoor/shared-types";

export interface SDKClientConfig {
  baseUrl: string;
  token?: string;
  tenantId?: string;
  clientName?: string;
}

export interface ListTenantsParams extends PaginationQuery {
  status?: string;
}

export interface ListUsersParams extends PaginationQuery {
  tenantId: string;
  status?: string;
}

export interface ListAccessKeysParams extends PaginationQuery {
  tenantId?: string;
  userId?: string;
  status?: string;
}

export interface ListNodesParams extends PaginationQuery {
  tenantId?: string;
  status?: string;
}

export interface ListAuditLogsParams extends PaginationQuery {
  tenantId?: string;
  action?: string;
  actorType?: string;
  targetType?: string;
  since?: string;
  until?: string;
}

export interface ListPoliciesParams extends PaginationQuery {
  tenantId?: string;
}

export interface ListUserOverridesParams extends PaginationQuery {
  tenantId: string;
  userId?: string;
}

export interface ListNodeCertificatesParams extends PaginationQuery {
  tenantId: string;
  nodeId: string;
  status?: "active" | "revoked";
}

export interface NodeProvisioningLookup {
  tenantId: string;
  nodeId?: string;
  nodeKeyId?: string;
}

interface APIErrorEnvelope {
  error?: {
    code?: string;
    message?: string;
    details?: Record<string, unknown>;
  };
}

export class SDKError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code = "request_failed",
    readonly requestId?: string,
    readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "SDKError";
  }
}

export class OpenerNetDoorClient {
  private readonly baseUrl: string;
  private readonly token?: string;
  private readonly tenantId?: string;
  private readonly clientName: string;

  constructor(config: SDKClientConfig) {
    this.baseUrl = config.baseUrl.trim().replace(/\/+$/, "");
    this.token = config.token;
    this.tenantId = config.tenantId;
    this.clientName = config.clientName ?? "opener-netdoor-sdk-ts/0.3.0";
  }

  health(): Promise<HealthResponse> {
    return this.request<HealthResponse>("GET", "/v1/health");
  }

  ready(): Promise<HealthResponse> {
    return this.request<HealthResponse>("GET", "/v1/ready");
  }

  listTenantsPage(params: ListTenantsParams = {}): Promise<TenantListResponse> {
    return this.request<TenantListResponse>("GET", "/v1/admin/tenants", {
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
      status: params.status,
    });
  }

  async listTenants(limit = 20, offset = 0): Promise<Tenant[]> {
    const response = await this.listTenantsPage({ limit, offset });
    return response.items;
  }

  createTenant(body: CreateTenantRequest): Promise<Tenant> {
    return this.request<Tenant>("POST", "/v1/admin/tenants", undefined, body);
  }

  listUsersPage(params: ListUsersParams): Promise<UserListResponse> {
    return this.request<UserListResponse>("GET", "/v1/admin/users", {
      tenant_id: params.tenantId,
      status: params.status,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  async listUsers(tenantId: string, limit = 20, offset = 0): Promise<User[]> {
    const response = await this.listUsersPage({ tenantId, limit, offset });
    return response.items;
  }

  createUser(body: CreateUserRequest): Promise<User> {
    return this.request<User>("POST", "/v1/admin/users", undefined, body);
  }

  blockUser(body: UserLifecycleRequest): Promise<User> {
    return this.request<User>("POST", "/v1/admin/users/block", undefined, body);
  }

  unblockUser(body: UserLifecycleRequest): Promise<User> {
    return this.request<User>("POST", "/v1/admin/users/unblock", undefined, body);
  }

  deleteUser(id: string, tenantId?: string): Promise<{ deleted: boolean; id: string; tenant_id: string }> {
    return this.request<{ deleted: boolean; id: string; tenant_id: string }>("DELETE", "/v1/admin/users", {
      id,
      tenant_id: tenantId ?? this.tenantId,
    });
  }

  listAccessKeysPage(params: ListAccessKeysParams = {}): Promise<AccessKeyListResponse> {
    return this.request<AccessKeyListResponse>("GET", "/v1/admin/access-keys", {
      tenant_id: params.tenantId ?? this.tenantId,
      user_id: params.userId,
      status: params.status,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  async listAccessKeys(params: ListAccessKeysParams = {}): Promise<AccessKey[]> {
    const response = await this.listAccessKeysPage(params);
    return response.items;
  }

  createAccessKey(body: CreateAccessKeyRequest): Promise<AccessKey> {
    return this.request<AccessKey>("POST", "/v1/admin/access-keys", undefined, body);
  }

  revokeAccessKey(id: string, tenantId?: string): Promise<AccessKey> {
    return this.request<AccessKey>("DELETE", "/v1/admin/access-keys", {
      id,
      tenant_id: tenantId ?? this.tenantId,
    });
  }

  async listTenantPoliciesPage(params: ListPoliciesParams = {}): Promise<TenantPolicyListResponse> {
    const response = await this.request<TenantPolicy | TenantPolicyListResponse>("GET", "/v1/admin/policies/tenants", {
      tenant_id: params.tenantId,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
    return normalizePolicyList(response);
  }

  async getTenantPolicy(tenantId: string): Promise<TenantPolicy | null> {
    const response = await this.request<TenantPolicy | TenantPolicyListResponse>("GET", "/v1/admin/policies/tenants", {
      tenant_id: tenantId,
      limit: 1,
      offset: 0,
    });
    if ("items" in response) {
      return response.items.at(0) ?? null;
    }
    return response;
  }

  upsertTenantPolicy(request: SetTenantPolicyRequest): Promise<TenantPolicy> {
    return this.request<TenantPolicy>("PUT", "/v1/admin/policies/tenants", undefined, request);
  }

  async listUserPolicyOverridesPage(params: ListUserOverridesParams): Promise<UserPolicyOverrideListResponse> {
    const response = await this.request<UserPolicyOverride | UserPolicyOverrideListResponse>("GET", "/v1/admin/policies/users", {
      tenant_id: params.tenantId,
      user_id: params.userId,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
    if ("items" in response) {
      return response;
    }
    return {
      items: [response],
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    };
  }

  async getUserPolicyOverride(tenantId: string, userId: string): Promise<UserPolicyOverride | null> {
    const response = await this.request<UserPolicyOverride | UserPolicyOverrideListResponse>("GET", "/v1/admin/policies/users", {
      tenant_id: tenantId,
      user_id: userId,
      limit: 1,
      offset: 0,
    });
    if ("items" in response) {
      return response.items.at(0) ?? null;
    }
    return response;
  }

  upsertUserPolicyOverride(request: SetUserPolicyOverrideRequest): Promise<UserPolicyOverride> {
    return this.request<UserPolicyOverride>("PUT", "/v1/admin/policies/users", undefined, request);
  }

  getEffectivePolicy(tenantId: string, userId: string): Promise<EffectivePolicy> {
    return this.request<EffectivePolicy>("GET", "/v1/admin/policies/effective", {
      tenant_id: tenantId,
      user_id: userId,
    });
  }

  registerDevice(request: RegisterDeviceRequest): Promise<Device> {
    return this.request<Device>("POST", "/v1/admin/devices/register", undefined, request);
  }

  listNodesPage(params: ListNodesParams = {}): Promise<NodeListResponse> {
    return this.request<NodeListResponse>("GET", "/v1/admin/nodes", {
      tenant_id: params.tenantId ?? this.tenantId,
      status: params.status,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  async listNodes(limit = 20, offset = 0): Promise<Node[]> {
    const response = await this.listNodesPage({ limit, offset });
    return response.items;
  }

  createNode(request: CreateNodeRequest): Promise<Node> {
    return this.request<Node>("POST", "/v1/admin/nodes", undefined, request);
  }

  getNodeDetail(nodeId: string, tenantId?: string): Promise<Node> {
    return this.request<Node>("GET", "/v1/admin/nodes/detail", {
      tenant_id: tenantId ?? this.tenantId,
      node_id: nodeId,
    });
  }

  revokeNode(request: NodeLifecycleRequest): Promise<Node> {
    return this.request<Node>("POST", "/v1/admin/nodes/revoke", undefined, request);
  }

  reactivateNode(request: NodeLifecycleRequest): Promise<Node> {
    return this.request<Node>("POST", "/v1/admin/nodes/reactivate", undefined, request);
  }

  getNodeProvisioning(params: NodeProvisioningLookup): Promise<NodeProvisioningContract> {
    return this.request<NodeProvisioningContract>("GET", "/v1/admin/nodes/provisioning", {
      tenant_id: params.tenantId,
      node_id: params.nodeId,
      node_key_id: params.nodeKeyId,
    });
  }

  listNodeCertificatesPage(params: ListNodeCertificatesParams): Promise<NodeCertificateListResponse> {
    return this.request<NodeCertificateListResponse>("GET", "/v1/admin/nodes/certificates", {
      tenant_id: params.tenantId,
      node_id: params.nodeId,
      status: params.status,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  rotateNodeCertificate(request: RotateNodeCertificateRequest): Promise<NodeCertificate> {
    return this.request<NodeCertificate>("POST", "/v1/admin/nodes/certificates/rotate", undefined, request);
  }

  revokeNodeCertificate(request: RevokeNodeCertificateRequest): Promise<NodeCertificate> {
    return this.request<NodeCertificate>("POST", "/v1/admin/nodes/certificates/revoke", undefined, request);
  }

  renewNodeCertificate(request: RenewNodeCertificateRequest): Promise<RenewNodeCertificateResult> {
    return this.request<RenewNodeCertificateResult>("POST", "/v1/admin/nodes/certificates/renew", undefined, request);
  }

  listPKIIssuersPage(params: PaginationQuery & { source?: "file" | "external"; status?: string } = {}): Promise<PKIIssuerListResponse> {
    return this.request<PKIIssuerListResponse>("GET", "/v1/admin/pki/issuers", {
      source: params.source,
      status: params.status,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  async listPKIIssuers(limit = 20, offset = 0): Promise<PKIIssuer[]> {
    const response = await this.listPKIIssuersPage({ limit, offset });
    return response.items;
  }

  createPKIIssuer(request: CreatePKIIssuerRequest): Promise<PKIIssuer> {
    return this.request<PKIIssuer>("POST", "/v1/admin/pki/issuers", undefined, request);
  }

  activatePKIIssuer(request: ActivatePKIIssuerRequest): Promise<CARotationResult> {
    return this.request<CARotationResult>("POST", "/v1/admin/pki/issuers/activate", undefined, request);
  }

  listAuditLogs(params: ListAuditLogsParams = {}): Promise<AuditLogListResponse> {
    return this.request<AuditLogListResponse>("GET", "/v1/admin/audit/logs", {
      tenant_id: params.tenantId ?? this.tenantId,
      action: params.action,
      actor_type: params.actorType,
      target_type: params.targetType,
      since: params.since,
      until: params.until,
      limit: params.limit ?? 20,
      offset: params.offset ?? 0,
    });
  }

  opsSnapshot(tenantId?: string): Promise<OpsSnapshot> {
    return this.request<OpsSnapshot>("GET", "/v1/admin/ops/snapshot", {
      tenant_id: tenantId ?? this.tenantId,
    });
  }

  private async request<T>(
    method: "GET" | "POST" | "PUT" | "DELETE",
    path: string,
    query?: Record<string, unknown>,
    body?: unknown,
  ): Promise<T> {
    const url = new URL(`${this.baseUrl}${path}`);
    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined && value !== null && `${value}` !== "") {
          url.searchParams.set(key, String(value));
        }
      }
    }

    const headers: Record<string, string> = {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Opener-Client": this.clientName,
      ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
    };

    const response = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });

    if (!response.ok) {
      const text = await response.text();
      const requestId = response.headers.get("X-Request-Id") ?? undefined;
      let parsed: APIErrorEnvelope | undefined;
      try {
        parsed = text ? (JSON.parse(text) as APIErrorEnvelope) : undefined;
      } catch {
        parsed = undefined;
      }

      const message =
        parsed?.error?.message ??
        (text.length > 0 ? text : `request failed with status ${response.status}`);
      throw new SDKError(
        message,
        response.status,
        parsed?.error?.code ?? "request_failed",
        requestId,
        parsed?.error?.details,
      );
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return (await response.json()) as T;
  }
}

function normalizePolicyList(input: TenantPolicy | TenantPolicyListResponse): TenantPolicyListResponse {
  if ("items" in input) {
    return input;
  }
  return {
    items: [input],
    limit: 20,
    offset: 0,
  };
}

export * from "./server_ops_adapter";



