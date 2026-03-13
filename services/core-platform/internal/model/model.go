package model

import "time"

type ActorPrincipal struct {
	Subject  string
	Scopes   []string
	TenantID string
}

func (a ActorPrincipal) IsPlatformAdmin() bool {
	for _, s := range a.Scopes {
		if s == "platform:admin" {
			return true
		}
	}
	return false
}

func (a ActorPrincipal) CanAccessTenant(tenantID string) bool {
	if tenantID == "" {
		return a.IsPlatformAdmin() || a.TenantID == ""
	}
	if a.IsPlatformAdmin() || a.TenantID == "" {
		return true
	}
	return a.TenantID == tenantID
}

type ListQuery struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Status string `json:"status,omitempty"`
}

type ListUsersQuery struct {
	ListQuery
	TenantID string `json:"tenant_id"`
}

type ListAccessKeysQuery struct {
	ListQuery
	TenantID string `json:"tenant_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
}

type GetUserSubscriptionQuery struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Format   string `json:"format,omitempty"`
}

type ListTenantPoliciesQuery struct {
	ListQuery
	TenantID string `json:"tenant_id,omitempty"`
}

type ListUserPolicyOverridesQuery struct {
	ListQuery
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id,omitempty"`
}

type GetEffectivePolicyQuery struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
}

type ListNodesQuery struct {
	ListQuery
	TenantID string `json:"tenant_id,omitempty"`
	Status   string `json:"status,omitempty"`
}

type ListNodeCertificatesQuery struct {
	ListQuery
	TenantID string `json:"tenant_id"`
	NodeID   string `json:"node_id"`
	Status   string `json:"status,omitempty"`
}

type ListPKIIssuersQuery struct {
	ListQuery
	Source string `json:"source,omitempty"`
	Status string `json:"status,omitempty"`
}

type ListAuditLogsQuery struct {
	ListQuery
	TenantID   string     `json:"tenant_id,omitempty"`
	Action     string     `json:"action,omitempty"`
	ActorType  string     `json:"actor_type,omitempty"`
	TargetType string     `json:"target_type,omitempty"`
	Since      *time.Time `json:"since,omitempty"`
	Until      *time.Time `json:"until,omitempty"`
}

type GetNodeProvisioningQuery struct {
	TenantID  string `json:"tenant_id"`
	NodeID    string `json:"node_id,omitempty"`
	NodeKeyID string `json:"node_key_id,omitempty"`
}

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email,omitempty"`
	Status    string    `json:"status"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessKey struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	UserID        string     `json:"user_id"`
	KeyType       string     `json:"key_type"`
	SecretRef     string     `json:"secret_ref"`
	ConnectionURI string     `json:"connection_uri,omitempty"`
	Status        string     `json:"status"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type SubscriptionConfig struct {
	ServerID string `json:"server_id"`
	Hostname string `json:"hostname"`
	Region   string `json:"region"`
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
	URI      string `json:"uri"`
}

type UserSubscription struct {
	TenantID        string               `json:"tenant_id"`
	UserID          string               `json:"user_id"`
	GeneratedAt     time.Time            `json:"generated_at"`
	Format          string               `json:"format"`
	SubscriptionURL string               `json:"subscription_url,omitempty"`
	Payload         string               `json:"payload"`
	ConfigCount     int                  `json:"config_count"`
	Configs         []SubscriptionConfig `json:"configs"`
}

type Device struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	UserID            string    `json:"user_id"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	Platform          string    `json:"platform"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

type Node struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenant_id"`
	Region              string     `json:"region"`
	Hostname            string     `json:"hostname"`
	NodeKeyID           string     `json:"node_key_id"`
	NodePublicKey       string     `json:"node_public_key"`
	ContractVersion     string     `json:"contract_version"`
	AgentVersion        string     `json:"agent_version"`
	Capabilities        []string   `json:"capabilities"`
	IdentityFingerprint string     `json:"identity_fingerprint"`
	Status              string     `json:"status"`
	LastSeenAt          *time.Time `json:"last_seen_at,omitempty"`
	LastHeartbeatAt     *time.Time `json:"last_heartbeat_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}


type NodeRuntime struct {
	NodeID               string     `json:"node_id"`
	TenantID             string     `json:"tenant_id"`
	RuntimeBackend       string     `json:"runtime_backend"`
	RuntimeProtocol      string     `json:"runtime_protocol"`
	ListenPort           int        `json:"listen_port"`
	RealityPublicKey     string     `json:"reality_public_key"`
	RealityShortID       string     `json:"reality_short_id"`
	RealityServerName    string     `json:"reality_server_name"`
	AppliedConfigVersion int        `json:"applied_config_version"`
	RuntimeStatus        string     `json:"runtime_status"`
	LastAppliedAt        *time.Time `json:"last_applied_at,omitempty"`
	LastError            string     `json:"last_error,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type RuntimeRevision struct {
	ID         int64      `json:"id"`
	NodeID     string     `json:"node_id"`
	TenantID   string     `json:"tenant_id"`
	Version    int        `json:"version"`
	ConfigJSON string     `json:"config_json"`
	Applied    bool       `json:"applied"`
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type RuntimeApplyRequest struct {
	TenantID string `json:"tenant_id"`
	NodeID   string `json:"node_id"`
}

type RuntimeConfigResponse struct {
	Node       Node            `json:"node"`
	Runtime    NodeRuntime     `json:"runtime"`
	Revision   RuntimeRevision `json:"revision"`
	ConfigJSON string          `json:"config_json"`
}
type NodeTLSIdentity struct {
	SerialNumber string `json:"serial_number"`
	CertPEM      string `json:"cert_pem,omitempty"`
}

type NodeCertificate struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	NodeID           string     `json:"node_id"`
	SerialNumber     string     `json:"serial_number"`
	CertPEM          string     `json:"cert_pem"`
	CAID             string     `json:"ca_id"`
	IssuerID         string     `json:"issuer_id,omitempty"`
	Issuer           string     `json:"issuer"`
	NotBefore        time.Time  `json:"not_before"`
	NotAfter         time.Time  `json:"not_after"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RotateFromCertID *string    `json:"rotate_from_cert_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type PKIIssuer struct {
	ID                 string         `json:"id"`
	IssuerID           string         `json:"issuer_id"`
	Source             string         `json:"source"`
	CAID               string         `json:"ca_id"`
	IssuerName         string         `json:"issuer_name"`
	CACertPEM          string         `json:"ca_cert_pem"`
	Status             string         `json:"status"`
	ActivatedAt        *time.Time     `json:"activated_at,omitempty"`
	RetiredAt          *time.Time     `json:"retired_at,omitempty"`
	RotateFromIssuerID *string        `json:"rotate_from_issuer_id,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
}

type CARotationResult struct {
	ActiveIssuer   PKIIssuer  `json:"active_issuer"`
	PreviousIssuer *PKIIssuer `json:"previous_issuer,omitempty"`
	RotatedAt      time.Time  `json:"rotated_at"`
}

type AuditLogRecord struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id,omitempty"`
	ActorType  string         `json:"actor_type"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type OpsNodeStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type OpsSnapshot struct {
	TenantID             string               `json:"tenant_id,omitempty"`
	GeneratedAt          time.Time            `json:"generated_at"`
	NodeStatus           []OpsNodeStatusCount `json:"node_status"`
	ActiveCertificates   int                  `json:"active_certificates"`
	ExpiringCertificates int                  `json:"expiring_certificates_24h"`
	TrafficBytes24h      int64                `json:"traffic_bytes_24h"`
	ReplayRejected24h    int                  `json:"replay_rejected_24h"`
	InvalidSignature24h  int                  `json:"invalid_signature_24h"`
}

type OpsTrafficPoint struct {
	TsHour     time.Time `json:"ts_hour"`
	BytesIn    int64     `json:"bytes_in"`
	BytesOut   int64     `json:"bytes_out"`
	BytesTotal int64     `json:"bytes_total"`
}

type OpsUserGrowthPoint struct {
	Day        string `json:"day"`
	NewUsers   int    `json:"new_users"`
	TotalUsers int    `json:"total_users"`
}

type OpsProtocolUsagePoint struct {
	Protocol   string `json:"protocol"`
	BytesTotal int64  `json:"bytes_total"`
}

type OpsTopServerPoint struct {
	NodeID      string `json:"node_id"`
	Hostname    string `json:"hostname"`
	Region      string `json:"region"`
	BytesTotal  int64  `json:"bytes_total"`
	LoadPercent int    `json:"load_percent"`
}

type OpsAnalytics struct {
	TenantID         string                  `json:"tenant_id,omitempty"`
	GeneratedAt      time.Time               `json:"generated_at"`
	TotalUsers       int                     `json:"total_users"`
	ActiveUsers      int                     `json:"active_users"`
	ActiveKeys       int                     `json:"active_keys"`
	OnlineServers    int                     `json:"online_servers"`
	TrafficBytes24h  int64                   `json:"traffic_bytes_24h"`
	TrafficHistory7d []OpsTrafficPoint       `json:"traffic_history_7d"`
	UserGrowth7d     []OpsUserGrowthPoint    `json:"user_growth_7d"`
	ProtocolUsage24h []OpsProtocolUsagePoint `json:"protocol_usage_24h"`
	TopServersByLoad []OpsTopServerPoint     `json:"top_servers_by_load"`
}

type NodeRegistrationResult struct {
	Node         Node                     `json:"node"`
	Provisioning NodeProvisioningContract `json:"provisioning"`
}

type NodeProvisioningContract struct {
	TenantID                 string   `json:"tenant_id"`
	NodeID                   string   `json:"node_id"`
	NodeKeyID                string   `json:"node_key_id"`
	ContractVersion          string   `json:"contract_version"`
	HeartbeatIntervalSeconds int      `json:"heartbeat_interval_seconds"`
	StaleAfterSeconds        int      `json:"stale_after_seconds"`
	OfflineAfterSeconds      int      `json:"offline_after_seconds"`
	RequiredCapabilities     []string `json:"required_capabilities"`
	NodeCertificateSerial    string   `json:"node_certificate_serial,omitempty"`
	NodeCertificatePEM       string   `json:"node_certificate_pem,omitempty"`
	NodePrivateKeyPEM        string   `json:"node_private_key_pem,omitempty"`
	NodeCertificateNotAfter  string   `json:"node_certificate_not_after,omitempty"`
}

type TenantPolicy struct {
	TenantID             string    `json:"tenant_id"`
	TrafficQuotaBytes    *int64    `json:"traffic_quota_bytes,omitempty"`
	DeviceLimit          *int      `json:"device_limit,omitempty"`
	DefaultKeyTTLSeconds *int      `json:"default_key_ttl_seconds,omitempty"`
	UpdatedBy            string    `json:"updated_by,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type UserPolicyOverride struct {
	TenantID          string    `json:"tenant_id"`
	UserID            string    `json:"user_id"`
	TrafficQuotaBytes *int64    `json:"traffic_quota_bytes,omitempty"`
	DeviceLimit       *int      `json:"device_limit,omitempty"`
	KeyTTLSeconds     *int      `json:"key_ttl_seconds,omitempty"`
	UpdatedBy         string    `json:"updated_by,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type EffectivePolicy struct {
	TenantID          string `json:"tenant_id"`
	UserID            string `json:"user_id"`
	TrafficQuotaBytes *int64 `json:"traffic_quota_bytes,omitempty"`
	DeviceLimit       *int   `json:"device_limit,omitempty"`
	KeyTTLSeconds     *int   `json:"key_ttl_seconds,omitempty"`
	UsageBytes        int64  `json:"usage_bytes"`
	QuotaExceeded     bool   `json:"quota_exceeded"`
	Source            string `json:"source"`
}

type CreateTenantRequest struct {
	Name string `json:"name"`
}

type CreateUserRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Note     string `json:"note,omitempty"`
}

type UserLifecycleRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
}

type CreateAccessKeyRequest struct {
	TenantID  string     `json:"tenant_id"`
	UserID    string     `json:"user_id"`
	KeyType   string     `json:"key_type"`
	SecretRef string     `json:"secret_ref,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type RevokeAccessKeyRequest struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id,omitempty"`
}

type SetTenantPolicyRequest struct {
	TenantID             string `json:"tenant_id"`
	TrafficQuotaBytes    *int64 `json:"traffic_quota_bytes,omitempty"`
	DeviceLimit          *int   `json:"device_limit,omitempty"`
	DefaultKeyTTLSeconds *int   `json:"default_key_ttl_seconds,omitempty"`
}

type SetUserPolicyOverrideRequest struct {
	TenantID          string `json:"tenant_id"`
	UserID            string `json:"user_id"`
	TrafficQuotaBytes *int64 `json:"traffic_quota_bytes,omitempty"`
	DeviceLimit       *int   `json:"device_limit,omitempty"`
	KeyTTLSeconds     *int   `json:"key_ttl_seconds,omitempty"`
}

type RegisterDeviceRequest struct {
	TenantID          string `json:"tenant_id"`
	UserID            string `json:"user_id"`
	DeviceFingerprint string `json:"device_fingerprint"`
	Platform          string `json:"platform"`
}

type RegisterNodeRequest struct {
	TenantID        string           `json:"tenant_id"`
	Region          string           `json:"region"`
	Hostname        string           `json:"hostname"`
	NodeKeyID       string           `json:"node_key_id"`
	NodePublicKey   string           `json:"node_public_key"`
	ContractVersion string           `json:"contract_version"`
	AgentVersion    string           `json:"agent_version"`
	Capabilities    []string         `json:"capabilities"`
	TLSIdentity     *NodeTLSIdentity `json:"tls_identity,omitempty"`
	Nonce           string           `json:"nonce"`
	SignedAt        int64            `json:"signed_at"`
	Signature       string           `json:"signature"`
}

type NodeHeartbeatRequest struct {
	TenantID        string           `json:"tenant_id"`
	NodeID          string           `json:"node_id"`
	NodeKeyID       string           `json:"node_key_id"`
	ContractVersion string           `json:"contract_version"`
	AgentVersion    string           `json:"agent_version"`
	TLSIdentity     *NodeTLSIdentity `json:"tls_identity,omitempty"`
	Nonce           string           `json:"nonce"`
	SignedAt        int64            `json:"signed_at"`
	Signature       string           `json:"signature"`
}

type NodeLifecycleRequest struct {
	TenantID string `json:"tenant_id"`
	NodeID   string `json:"node_id"`
}

type CreateNodeRequest struct {
	TenantID        string   `json:"tenant_id"`
	Region          string   `json:"region"`
	Hostname        string   `json:"hostname"`
	AgentVersion    string   `json:"agent_version,omitempty"`
	ContractVersion string   `json:"contract_version,omitempty"`
	Capabilities    []string `json:"capabilities,omitempty"`
}

type IssueNodeCertificateRequest struct {
	TenantID         string    `json:"tenant_id"`
	NodeID           string    `json:"node_id"`
	SerialNumber     string    `json:"serial_number"`
	CertPEM          string    `json:"cert_pem"`
	CAID             string    `json:"ca_id"`
	IssuerID         string    `json:"issuer_id,omitempty"`
	Issuer           string    `json:"issuer"`
	NotBefore        time.Time `json:"not_before"`
	NotAfter         time.Time `json:"not_after"`
	RotateFromCertID *string   `json:"rotate_from_cert_id,omitempty"`
}

type RotateNodeCertificateRequest struct {
	TenantID string `json:"tenant_id"`
	NodeID   string `json:"node_id"`
}

type RevokeNodeCertificateRequest struct {
	TenantID       string `json:"tenant_id"`
	NodeID         string `json:"node_id"`
	CertificateID  string `json:"certificate_id,omitempty"`
	SerialNumber   string `json:"serial_number,omitempty"`
	RevocationNote string `json:"revocation_note,omitempty"`
}

type RenewNodeCertificateRequest struct {
	TenantID   string `json:"tenant_id"`
	NodeID     string `json:"node_id"`
	Force      bool   `json:"force,omitempty"`
	TTLSeconds *int   `json:"ttl_seconds,omitempty"`
}

type RenewNodeCertificateResult struct {
	TenantID              string          `json:"tenant_id"`
	NodeID                string          `json:"node_id"`
	PreviousCertificateID string          `json:"previous_certificate_id,omitempty"`
	PreviousSerialNumber  string          `json:"previous_serial_number,omitempty"`
	Certificate           NodeCertificate `json:"certificate"`
	Renewed               bool            `json:"renewed"`
}

type CreatePKIIssuerRequest struct {
	IssuerID   string         `json:"issuer_id"`
	Source     string         `json:"source,omitempty"`
	CAID       string         `json:"ca_id,omitempty"`
	IssuerName string         `json:"issuer_name,omitempty"`
	CACertPEM  string         `json:"ca_cert_pem,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Activate   bool           `json:"activate,omitempty"`
}

type ActivatePKIIssuerRequest struct {
	IssuerID string `json:"issuer_id"`
}

type RetirePKIIssuerRequest struct {
	IssuerID string `json:"issuer_id"`
}

type ConsumeNodeNonceRequest struct {
	TenantID    string    `json:"tenant_id"`
	NodeKeyID   string    `json:"node_key_id"`
	RequestType string    `json:"request_type"`
	Nonce       string    `json:"nonce"`
	SignedAt    time.Time `json:"signed_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type AuditLogEvent struct {
	TenantID   string         `json:"tenant_id"`
	ActorType  string         `json:"actor_type"`
	ActorSub   string         `json:"actor_sub,omitempty"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	OccurredAt time.Time      `json:"occurred_at,omitempty"`
}




