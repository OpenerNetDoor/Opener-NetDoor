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
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	UserID    string     `json:"user_id"`
	KeyType   string     `json:"key_type"`
	SecretRef string     `json:"secret_ref"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
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
	TenantID        string   `json:"tenant_id"`
	Region          string   `json:"region"`
	Hostname        string   `json:"hostname"`
	NodeKeyID       string   `json:"node_key_id"`
	NodePublicKey   string   `json:"node_public_key"`
	ContractVersion string   `json:"contract_version"`
	AgentVersion    string   `json:"agent_version"`
	Capabilities    []string `json:"capabilities"`
	SignedAt        int64    `json:"signed_at"`
	Signature       string   `json:"signature"`
}

type NodeHeartbeatRequest struct {
	TenantID        string `json:"tenant_id"`
	NodeID          string `json:"node_id"`
	NodeKeyID       string `json:"node_key_id"`
	ContractVersion string `json:"contract_version"`
	AgentVersion    string `json:"agent_version"`
	SignedAt        int64  `json:"signed_at"`
	Signature       string `json:"signature"`
}
