export type Scope =
  | "admin:read"
  | "admin:write"
  | "platform:admin"
  | "user:read"
  | "user:write";

export type ThemeMode = "light" | "dark" | "system";

export interface SessionTokenPayload {
  sub: string;
  tenant_id?: string;
  scopes: Scope[];
  exp: number;
}

export interface PaginationQuery {
  limit?: number;
  offset?: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  limit: number;
  offset: number;
}

export interface HealthResponse {
  status: string;
}

export interface Tenant {
  id: string;
  name: string;
  status: "active" | "suspended" | string;
  created_at: string;
}

export type TenantListResponse = PaginatedResponse<Tenant>;

export interface CreateTenantRequest {
  name: string;
}

export interface User {
  id: string;
  tenant_id: string;
  email?: string;
  status: "active" | "blocked" | string;
  note?: string;
  created_at: string;
}

export type UserListResponse = PaginatedResponse<User>;

export interface CreateUserRequest {
  tenant_id: string;
  email: string;
  note?: string;
}

export interface UserLifecycleRequest {
  tenant_id: string;
  user_id: string;
}

export interface AccessKey {
  id: string;
  tenant_id: string;
  user_id: string;
  key_type: string;
  status: "active" | "revoked" | string;
  secret_ref: string;
  connection_uri?: string;
  created_at: string;
  expires_at?: string;
}

export type AccessKeyListResponse = PaginatedResponse<AccessKey>;

export interface CreateAccessKeyRequest {
  tenant_id: string;
  user_id: string;
  key_type: string;
  secret_ref?: string;
  expires_at?: string;
}

export interface TenantPolicy {
  tenant_id: string;
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  default_key_ttl_seconds?: number | null;
  updated_by?: string;
  updated_at: string;
}

export type TenantPolicyListResponse = PaginatedResponse<TenantPolicy>;

export interface SetTenantPolicyRequest {
  tenant_id: string;
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  default_key_ttl_seconds?: number | null;
}

export interface UserPolicyOverride {
  tenant_id: string;
  user_id: string;
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  key_ttl_seconds?: number | null;
  updated_by?: string;
  updated_at: string;
}

export type UserPolicyOverrideListResponse = PaginatedResponse<UserPolicyOverride>;

export interface SetUserPolicyOverrideRequest {
  tenant_id: string;
  user_id: string;
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  key_ttl_seconds?: number | null;
}

export interface EffectivePolicy {
  tenant_id: string;
  user_id: string;
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  key_ttl_seconds?: number | null;
  usage_bytes: number;
  quota_exceeded: boolean;
  source: "tenant_default" | "user_override";
}

export interface RegisterDeviceRequest {
  tenant_id: string;
  user_id: string;
  device_fingerprint: string;
  platform: string;
}

export interface Device {
  id: string;
  tenant_id: string;
  user_id: string;
  device_fingerprint: string;
  platform: string;
  status: "active" | "blocked" | string;
  created_at: string;
}

export interface NodeTLSIdentity {
  serial_number: string;
  cert_pem?: string;
}

export interface Node {
  id: string;
  tenant_id: string;
  node_key_id: string;
  node_public_key?: string;
  hostname: string;
  region: string;
  contract_version?: string;
  agent_version?: string;
  capabilities?: string[];
  identity_fingerprint?: string;
  status: "pending" | "active" | "stale" | "offline" | "revoked" | string;
  last_seen_at?: string;
  last_heartbeat_at?: string;
  created_at: string;
}

export interface NodeRuntime {
  node_id: string;
  tenant_id: string;
  runtime_backend: string;
  runtime_protocol: string;
  listen_port: number;
  reality_public_key: string;
  reality_short_id: string;
  reality_server_name: string;
  applied_config_version: number;
  runtime_status: string;
  last_applied_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface RuntimeRevision {
  id: number;
  node_id: string;
  tenant_id: string;
  version: number;
  config_json: string;
  applied: boolean;
  applied_at?: string;
  created_at: string;
}

export interface RuntimeConfigResponse {
  node: Node;
  runtime: NodeRuntime;
  revision: RuntimeRevision;
  config_json: string;
}

export interface RuntimeApplyRequest {
  tenant_id: string;
  node_id: string;
}
export type NodeListResponse = PaginatedResponse<Node>;

export interface RegisterNodeRequest {
  tenant_id: string;
  region: string;
  hostname: string;
  node_key_id: string;
  node_public_key: string;
  contract_version: string;
  agent_version: string;
  capabilities: string[];
  tls_identity?: NodeTLSIdentity;
  nonce: string;
  signed_at: number;
  signature: string;
}

export interface NodeHeartbeatRequest {
  tenant_id: string;
  node_id: string;
  node_key_id: string;
  contract_version: string;
  agent_version: string;
  tls_identity?: NodeTLSIdentity;
  nonce: string;
  signed_at: number;
  signature: string;
}

export interface NodeLifecycleRequest {
  tenant_id: string;
  node_id: string;
}

export interface CreateNodeRequest {
  tenant_id: string;
  region: string;
  hostname: string;
  agent_version?: string;
  contract_version?: string;
  capabilities?: string[];
}

export interface NodeProvisioningContract {
  tenant_id: string;
  node_id: string;
  node_key_id: string;
  contract_version: string;
  heartbeat_interval_seconds: number;
  stale_after_seconds: number;
  offline_after_seconds: number;
  required_capabilities: string[];
  node_certificate_serial?: string;
  node_certificate_pem?: string;
  node_private_key_pem?: string;
  node_certificate_not_after?: string;
}

export interface NodeRegistrationResult {
  node: Node;
  provisioning: NodeProvisioningContract;
}

export interface NodeCertificate {
  id: string;
  tenant_id: string;
  node_id: string;
  serial_number: string;
  cert_pem: string;
  ca_id: string;
  issuer_id?: string;
  issuer: string;
  not_before: string;
  not_after: string;
  revoked_at?: string | null;
  rotate_from_cert_id?: string | null;
  created_at: string;
}

export type NodeCertificateListResponse = PaginatedResponse<NodeCertificate>;

export interface RotateNodeCertificateRequest {
  tenant_id: string;
  node_id: string;
}

export interface RevokeNodeCertificateRequest {
  tenant_id: string;
  node_id: string;
  certificate_id?: string;
  serial_number?: string;
  revocation_note?: string;
}

export interface RenewNodeCertificateRequest {
  tenant_id: string;
  node_id: string;
  force?: boolean;
  ttl_seconds?: number;
}

export interface RenewNodeCertificateResult {
  tenant_id: string;
  node_id: string;
  previous_certificate_id?: string;
  previous_serial_number?: string;
  certificate: NodeCertificate;
  renewed: boolean;
}

export interface PKIIssuer {
  id: string;
  issuer_id: string;
  source: "file" | "external";
  ca_id?: string;
  issuer_name?: string;
  ca_cert_pem?: string;
  status: "pending" | "active" | "retired";
  activated_at?: string | null;
  retired_at?: string | null;
  rotate_from_issuer_id?: string | null;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export type PKIIssuerListResponse = PaginatedResponse<PKIIssuer>;

export interface CreatePKIIssuerRequest {
  issuer_id: string;
  source?: "file" | "external";
  ca_id?: string;
  issuer_name?: string;
  ca_cert_pem?: string;
  activate?: boolean;
  metadata?: Record<string, unknown>;
}

export interface ActivatePKIIssuerRequest {
  issuer_id: string;
}

export interface CARotationResult {
  active_issuer: PKIIssuer;
  previous_issuer?: PKIIssuer;
  rotated_at: string;
}

export interface AuditLogRecord {
  id: string;
  tenant_id?: string;
  actor_type: string;
  action: string;
  target_type?: string;
  target_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export type AuditLogListResponse = PaginatedResponse<AuditLogRecord>;

export interface OpsNodeStatusCount {
  status: string;
  count: number;
}

export interface OpsSnapshot {
  tenant_id?: string;
  generated_at: string;
  node_status: OpsNodeStatusCount[];
  active_certificates: number;
  expiring_certificates_24h: number;
  traffic_bytes_24h: number;
  replay_rejected_24h: number;
  invalid_signature_24h: number;
}

export interface OpsTrafficPoint {
  ts_hour: string;
  bytes_in: number;
  bytes_out: number;
  bytes_total: number;
}

export interface OpsUserGrowthPoint {
  day: string;
  new_users: number;
  total_users: number;
}

export interface OpsProtocolUsagePoint {
  protocol: string;
  bytes_total: number;
}

export interface OpsTopServerPoint {
  node_id: string;
  hostname: string;
  region: string;
  bytes_total: number;
  load_percent: number;
}

export interface OpsAnalytics {
  tenant_id?: string;
  generated_at: string;
  total_users: number;
  active_users: number;
  active_keys: number;
  online_servers: number;
  traffic_bytes_24h: number;
  traffic_history_7d: OpsTrafficPoint[];
  user_growth_7d: OpsUserGrowthPoint[];
  protocol_usage_24h: OpsProtocolUsagePoint[];
  top_servers_by_load: OpsTopServerPoint[];
}
export type ConnectionState =
  | "idle"
  | "resolving_profile"
  | "connecting"
  | "connected"
  | "degraded"
  | "reconnecting"
  | "blocked_or_timeout"
  | "auth_failed"
  | "quota_or_policy_denied";

export type ProtocolFamily = "tunnel_vpn" | "proxy" | "udp_quic" | "special";

export type TransportWrapper = "tcp_tls" | "ws_tls" | "grpc_tls" | "h2" | "h3" | "reality";

export type ProtocolId =
  | "wireguard"
  | "openvpn"
  | "ipsec_ikev2"
  | "vless"
  | "vmess"
  | "trojan"
  | "shadowsocks"
  | "socks5"
  | "http_proxy"
  | "tuic"
  | "hysteria2"
  | "mieru"
  | "nieva";

export interface ProtocolCatalogEntry {
  id: ProtocolId;
  label: string;
  family: ProtocolFamily;
  support: "supported" | "unsupported_unverified";
  supports_cdn: boolean;
  supports_quic: boolean;
  supports_reality: boolean;
  compatible_wrappers: TransportWrapper[];
  notes: string;
}

export const PROTOCOL_CATALOG: ProtocolCatalogEntry[] = [
  {
    id: "wireguard",
    label: "WireGuard",
    family: "tunnel_vpn",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: [],
    notes: "UDP-native VPN tunnel with strong performance; not CDN-friendly.",
  },
  {
    id: "openvpn",
    label: "OpenVPN",
    family: "tunnel_vpn",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls"],
    notes: "Legacy fallback profile for broad compatibility.",
  },
  {
    id: "ipsec_ikev2",
    label: "IPsec/IKEv2",
    family: "tunnel_vpn",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: [],
    notes: "Native OS compatibility path on mobile and desktop.",
  },
  {
    id: "vless",
    label: "VLESS",
    family: "proxy",
    support: "supported",
    supports_cdn: true,
    supports_quic: false,
    supports_reality: true,
    compatible_wrappers: ["tcp_tls", "ws_tls", "grpc_tls", "h2", "reality"],
    notes: "Primary anti-censorship profile with WS/gRPC CDN-friendly modes.",
  },
  {
    id: "vmess",
    label: "VMess",
    family: "proxy",
    support: "supported",
    supports_cdn: true,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls", "ws_tls", "grpc_tls", "h2"],
    notes: "Legacy-compatible proxy profile without Reality support.",
  },
  {
    id: "trojan",
    label: "Trojan",
    family: "proxy",
    support: "supported",
    supports_cdn: true,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls", "ws_tls"],
    notes: "TLS-based fallback profile, WS mode runtime-dependent.",
  },
  {
    id: "shadowsocks",
    label: "Shadowsocks",
    family: "proxy",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls"],
    notes: "Plugin-dependent wrappers and compatibility modes.",
  },
  {
    id: "socks5",
    label: "SOCKS5",
    family: "proxy",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls"],
    notes: "Operational and diagnostics fallback proxy endpoint.",
  },
  {
    id: "http_proxy",
    label: "HTTP Proxy",
    family: "proxy",
    support: "supported",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: ["tcp_tls", "h2"],
    notes: "Legacy corporate-network compatibility path.",
  },
  {
    id: "tuic",
    label: "TUIC",
    family: "udp_quic",
    support: "supported",
    supports_cdn: false,
    supports_quic: true,
    supports_reality: false,
    compatible_wrappers: ["h3"],
    notes: "QUIC-native transport, high performance, usually not CDN-friendly.",
  },
  {
    id: "hysteria2",
    label: "Hysteria2",
    family: "udp_quic",
    support: "supported",
    supports_cdn: false,
    supports_quic: true,
    supports_reality: false,
    compatible_wrappers: ["h3"],
    notes: "QUIC-native protocol optimized for unstable networks.",
  },
  {
    id: "mieru",
    label: "mieru",
    family: "special",
    support: "supported",
    supports_cdn: false,
    supports_quic: true,
    supports_reality: false,
    compatible_wrappers: ["h3"],
    notes: "Advanced protocol family; treat as opt-in profile class.",
  },
  {
    id: "nieva",
    label: "Nieva",
    family: "special",
    support: "unsupported_unverified",
    supports_cdn: false,
    supports_quic: false,
    supports_reality: false,
    compatible_wrappers: [],
    notes: "Unverified protocol placeholder. Runtime support intentionally disabled.",
  },
];

export interface ProfilePolicy {
  traffic_quota_bytes?: number | null;
  device_limit?: number | null;
  key_ttl_seconds?: number | null;
  enforce_strict_tls: boolean;
  allow_quic_profiles: boolean;
}

export interface ServerBinding {
  node_id: string;
  region: string;
  endpoint: string;
  port: number;
  transport: TransportWrapper;
  sni?: string;
  cdn_front_domain?: string;
}

export interface ClientCapability {
  platform: "ios" | "android" | "windows" | "macos" | "linux" | "web";
  supports_quic: boolean;
  supports_wireguard: boolean;
  supports_split_tunnel: boolean;
  supports_always_on: boolean;
}

export interface ProtocolProfile {
  id: string;
  tenant_id: string;
  name: string;
  protocol: ProtocolId;
  transport: TransportWrapper;
  security_mode: "tls" | "reality" | "none";
  cdn_friendly: boolean;
  enabled: boolean;
  deprecated: boolean;
  policy: ProfilePolicy;
  bindings: ServerBinding[];
  created_at: string;
  updated_at: string;
}

export interface ProfileRevision {
  id: string;
  profile_id: string;
  revision: number;
  payload_hash: string;
  change_note?: string;
  changed_by: string;
  created_at: string;
}

export interface ProfileAssignment {
  id: string;
  tenant_id: string;
  user_id: string;
  profile_id: string;
  fallback_profile_ids: string[];
  auto_select_node: boolean;
  favorite_nodes: string[];
  created_at: string;
  updated_at: string;
}

export interface ServerInventoryItem {
  id: string;
  tenant_id?: string;
  region: string;
  ip: string;
  domain?: string;
  status: "active" | "needs_attention" | "installing" | "degraded" | "offline";
  provider?: string;
  created_at: string;
  updated_at: string;
}

export type ServerOperationType =
  | "install"
  | "upgrade"
  | "rollback"
  | "backup"
  | "restore"
  | "cert_rotate"
  | "firewall_apply"
  | "cdn_apply";

export interface ServerOperationPlan {
  server_id: string;
  operation: ServerOperationType;
  commands: string[];
  warnings: string[];
  requires_maintenance_window: boolean;
  generated_at: string;
  source: "live_api" | "local_planner";
}

export interface ServerOperationJob {
  id: string;
  server_id: string;
  operation: ServerOperationType;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at?: string;
  finished_at?: string;
  correlation_id: string;
  logs: Array<Record<string, unknown>>;
  source: "live_api" | "local_planner";
}

export interface ServerFilter {
  region?: string;
  status?: string;
  query?: string;
}

export interface ScopeMismatchInfo {
  expected_tenant_id?: string;
  actor_tenant_id?: string;
  required_scopes: Scope[];
}




export interface AdminSessionInfo {
  authenticated: boolean;
  subject: string;
  tenant_id: string;
  scopes: Scope[];
  expires_at: string;
}



