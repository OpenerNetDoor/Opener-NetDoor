package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "application error"
}

func (e *AppError) Unwrap() error { return e.Err }

type Options struct {
	NodeSigningSecret        string
	NodeContractVersion      string
	NodeHeartbeatInterval    time.Duration
	NodeStaleAfter           time.Duration
	NodeOfflineAfter         time.Duration
	NodeRequiredCapabilities []string
	NodeSignatureMaxSkew     time.Duration
}

func (o Options) withDefaults() Options {
	if strings.TrimSpace(o.NodeSigningSecret) == "" {
		o.NodeSigningSecret = "opener-netdoor-stage5-dev-signing-secret"
	}
	if strings.TrimSpace(o.NodeContractVersion) == "" {
		o.NodeContractVersion = "2026-03-10.stage5.v1"
	}
	if o.NodeHeartbeatInterval <= 0 {
		o.NodeHeartbeatInterval = 30 * time.Second
	}
	if o.NodeStaleAfter <= 0 {
		o.NodeStaleAfter = 90 * time.Second
	}
	if o.NodeOfflineAfter <= 0 {
		o.NodeOfflineAfter = 5 * time.Minute
	}
	if o.NodeOfflineAfter <= o.NodeStaleAfter {
		o.NodeOfflineAfter = o.NodeStaleAfter + 2*time.Minute
	}
	if len(o.NodeRequiredCapabilities) == 0 {
		o.NodeRequiredCapabilities = []string{"heartbeat.v1", "provisioning.v1"}
	}
	if o.NodeSignatureMaxSkew <= 0 {
		o.NodeSignatureMaxSkew = 5 * time.Minute
	}
	return o
}

type Service interface {
	Health(ctx context.Context) error

	ListTenants(ctx context.Context, actor model.ActorPrincipal, q model.ListQuery) ([]model.Tenant, error)
	CreateTenant(ctx context.Context, actor model.ActorPrincipal, in model.CreateTenantRequest) (model.Tenant, error)

	ListUsers(ctx context.Context, actor model.ActorPrincipal, q model.ListUsersQuery) ([]model.User, error)
	CreateUser(ctx context.Context, actor model.ActorPrincipal, in model.CreateUserRequest) (model.User, error)

	ListAccessKeys(ctx context.Context, actor model.ActorPrincipal, q model.ListAccessKeysQuery) ([]model.AccessKey, error)
	CreateAccessKey(ctx context.Context, actor model.ActorPrincipal, in model.CreateAccessKeyRequest) (model.AccessKey, error)
	RevokeAccessKey(ctx context.Context, actor model.ActorPrincipal, in model.RevokeAccessKeyRequest) (model.AccessKey, error)

	ListTenantPolicies(ctx context.Context, actor model.ActorPrincipal, q model.ListTenantPoliciesQuery) ([]model.TenantPolicy, error)
	GetTenantPolicy(ctx context.Context, actor model.ActorPrincipal, tenantID string) (model.TenantPolicy, error)
	SetTenantPolicy(ctx context.Context, actor model.ActorPrincipal, in model.SetTenantPolicyRequest) (model.TenantPolicy, error)

	ListUserPolicyOverrides(ctx context.Context, actor model.ActorPrincipal, q model.ListUserPolicyOverridesQuery) ([]model.UserPolicyOverride, error)
	GetUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, tenantID string, userID string) (model.UserPolicyOverride, error)
	SetUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, in model.SetUserPolicyOverrideRequest) (model.UserPolicyOverride, error)

	GetEffectivePolicy(ctx context.Context, actor model.ActorPrincipal, q model.GetEffectivePolicyQuery) (model.EffectivePolicy, error)
	RegisterDevice(ctx context.Context, actor model.ActorPrincipal, in model.RegisterDeviceRequest) (model.Device, error)

	ListNodes(ctx context.Context, actor model.ActorPrincipal, q model.ListNodesQuery) ([]model.Node, error)
	RegisterNode(ctx context.Context, actor model.ActorPrincipal, in model.RegisterNodeRequest) (model.NodeRegistrationResult, error)
	NodeHeartbeat(ctx context.Context, actor model.ActorPrincipal, in model.NodeHeartbeatRequest) (model.Node, error)
	GetNodeProvisioning(ctx context.Context, actor model.ActorPrincipal, q model.GetNodeProvisioningQuery) (model.NodeProvisioningContract, error)
}

type CoreService struct {
	store store.Store
	opts  Options
}

func New(s store.Store, opts ...Options) *CoreService {
	cfg := Options{}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	return &CoreService{store: s, opts: cfg.withDefaults()}
}

func (s *CoreService) Health(ctx context.Context) error {
	if err := s.store.Ping(ctx); err != nil {
		return &AppError{Status: 503, Code: "db_unavailable", Message: "database is unavailable", Err: err}
	}
	return nil
}

func (s *CoreService) ListTenants(ctx context.Context, actor model.ActorPrincipal, q model.ListQuery) ([]model.Tenant, error) {
	if actor.TenantID != "" && !actor.IsPlatformAdmin() {
		t, err := s.store.GetTenantByID(ctx, actor.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, &AppError{Status: 404, Code: "tenant_not_found", Message: "tenant not found", Err: err}
			}
			return nil, &AppError{Status: 500, Code: "tenant_list_failed", Message: "failed to list tenants", Err: err}
		}
		return []model.Tenant{t}, nil
	}
	items, err := s.store.ListTenants(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "tenant_list_failed", Message: "failed to list tenants", Err: err}
	}
	return items, nil
}

func (s *CoreService) CreateTenant(ctx context.Context, actor model.ActorPrincipal, in model.CreateTenantRequest) (model.Tenant, error) {
	if !actor.IsPlatformAdmin() {
		return model.Tenant{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot create tenants"}
	}
	if strings.TrimSpace(in.Name) == "" {
		return model.Tenant{}, &AppError{Status: 400, Code: "validation_error", Message: "name is required"}
	}
	item, err := s.store.CreateTenant(ctx, in)
	if err != nil {
		return model.Tenant{}, mapStoreError("tenant_create_failed", err)
	}
	return item, nil
}

func (s *CoreService) ListUsers(ctx context.Context, actor model.ActorPrincipal, q model.ListUsersQuery) ([]model.User, error) {
	if strings.TrimSpace(q.TenantID) == "" {
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	items, err := s.store.ListUsers(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "user_list_failed", Message: "failed to list users", Err: err}
	}
	return items, nil
}

func (s *CoreService) CreateUser(ctx context.Context, actor model.ActorPrincipal, in model.CreateUserRequest) (model.User, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.Email) == "" {
		return model.User{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and email are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.User{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	item, err := s.store.CreateUser(ctx, in)
	if err != nil {
		return model.User{}, mapStoreError("user_create_failed", err)
	}
	return item, nil
}

func (s *CoreService) ListAccessKeys(ctx context.Context, actor model.ActorPrincipal, q model.ListAccessKeysQuery) ([]model.AccessKey, error) {
	if !actor.IsPlatformAdmin() {
		if strings.TrimSpace(q.TenantID) == "" {
			q.TenantID = actor.TenantID
		}
		if !actor.CanAccessTenant(q.TenantID) {
			return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
		}
	}
	items, err := s.store.ListAccessKeys(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "access_key_list_failed", Message: "failed to list access keys", Err: err}
	}
	return items, nil
}

func (s *CoreService) CreateAccessKey(ctx context.Context, actor model.ActorPrincipal, in model.CreateAccessKeyRequest) (model.AccessKey, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.KeyType) == "" {
		return model.AccessKey{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id, user_id and key_type are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.AccessKey{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	effective, err := s.resolveEffectivePolicy(ctx, in.TenantID, in.UserID)
	if err != nil {
		return model.AccessKey{}, err
	}
	if err := s.enforceQuota(effective); err != nil {
		return model.AccessKey{}, err
	}

	if strings.TrimSpace(in.SecretRef) == "" {
		in.SecretRef = "secret://generated/" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	if in.ExpiresAt == nil && effective.KeyTTLSeconds != nil {
		ttl := *effective.KeyTTLSeconds
		expiresAt := time.Now().UTC().Add(time.Duration(ttl) * time.Second)
		in.ExpiresAt = &expiresAt
	}

	item, err := s.store.CreateAccessKey(ctx, in)
	if err != nil {
		return model.AccessKey{}, mapStoreError("access_key_create_failed", err)
	}
	return item, nil
}

func (s *CoreService) RevokeAccessKey(ctx context.Context, actor model.ActorPrincipal, in model.RevokeAccessKeyRequest) (model.AccessKey, error) {
	if strings.TrimSpace(in.ID) == "" {
		return model.AccessKey{}, &AppError{Status: 400, Code: "validation_error", Message: "id is required"}
	}
	tenantID := strings.TrimSpace(in.TenantID)
	if !actor.IsPlatformAdmin() {
		if tenantID == "" {
			tenantID = actor.TenantID
		}
		if !actor.CanAccessTenant(tenantID) {
			return model.AccessKey{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
		}
	}

	item, err := s.store.RevokeAccessKey(ctx, in.ID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AccessKey{}, &AppError{Status: 404, Code: "access_key_not_found", Message: "access key not found", Err: err}
		}
		return model.AccessKey{}, mapStoreError("access_key_revoke_failed", err)
	}
	return item, nil
}

func (s *CoreService) ListTenantPolicies(ctx context.Context, actor model.ActorPrincipal, q model.ListTenantPoliciesQuery) ([]model.TenantPolicy, error) {
	if actor.TenantID != "" && !actor.IsPlatformAdmin() {
		q.TenantID = actor.TenantID
	}
	if strings.TrimSpace(q.TenantID) != "" && !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	items, err := s.store.ListTenantPolicies(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "tenant_policy_list_failed", Message: "failed to list tenant policies", Err: err}
	}
	return items, nil
}

func (s *CoreService) GetTenantPolicy(ctx context.Context, actor model.ActorPrincipal, tenantID string) (model.TenantPolicy, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return model.TenantPolicy{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(tenantID) {
		return model.TenantPolicy{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	item, err := s.store.GetTenantPolicy(ctx, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.TenantPolicy{}, &AppError{Status: 404, Code: "tenant_policy_not_found", Message: "tenant policy not found", Err: err}
		}
		return model.TenantPolicy{}, &AppError{Status: 500, Code: "tenant_policy_get_failed", Message: "failed to get tenant policy", Err: err}
	}
	return item, nil
}

func (s *CoreService) SetTenantPolicy(ctx context.Context, actor model.ActorPrincipal, in model.SetTenantPolicyRequest) (model.TenantPolicy, error) {
	if strings.TrimSpace(in.TenantID) == "" {
		return model.TenantPolicy{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.TenantPolicy{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if err := validatePolicyValues(in.TrafficQuotaBytes, in.DeviceLimit, in.DefaultKeyTTLSeconds); err != nil {
		return model.TenantPolicy{}, err
	}
	item, err := s.store.UpsertTenantPolicy(ctx, actor, in)
	if err != nil {
		return model.TenantPolicy{}, mapStoreError("tenant_policy_set_failed", err)
	}
	return item, nil
}

func (s *CoreService) ListUserPolicyOverrides(ctx context.Context, actor model.ActorPrincipal, q model.ListUserPolicyOverridesQuery) ([]model.UserPolicyOverride, error) {
	if strings.TrimSpace(q.TenantID) == "" {
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	items, err := s.store.ListUserPolicyOverrides(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "user_policy_list_failed", Message: "failed to list user policy overrides", Err: err}
	}
	return items, nil
}

func (s *CoreService) GetUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, tenantID string, userID string) (model.UserPolicyOverride, error) {
	tenantID = strings.TrimSpace(tenantID)
	userID = strings.TrimSpace(userID)
	if tenantID == "" || userID == "" {
		return model.UserPolicyOverride{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and user_id are required"}
	}
	if !actor.CanAccessTenant(tenantID) {
		return model.UserPolicyOverride{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	item, err := s.store.GetUserPolicyOverride(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.UserPolicyOverride{}, &AppError{Status: 404, Code: "user_policy_not_found", Message: "user policy override not found", Err: err}
		}
		return model.UserPolicyOverride{}, &AppError{Status: 500, Code: "user_policy_get_failed", Message: "failed to get user policy override", Err: err}
	}
	return item, nil
}

func (s *CoreService) SetUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, in model.SetUserPolicyOverrideRequest) (model.UserPolicyOverride, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.UserID) == "" {
		return model.UserPolicyOverride{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and user_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.UserPolicyOverride{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if err := validatePolicyValues(in.TrafficQuotaBytes, in.DeviceLimit, in.KeyTTLSeconds); err != nil {
		return model.UserPolicyOverride{}, err
	}
	item, err := s.store.UpsertUserPolicyOverride(ctx, actor, in)
	if err != nil {
		return model.UserPolicyOverride{}, mapStoreError("user_policy_set_failed", err)
	}
	return item, nil
}

func (s *CoreService) GetEffectivePolicy(ctx context.Context, actor model.ActorPrincipal, q model.GetEffectivePolicyQuery) (model.EffectivePolicy, error) {
	if strings.TrimSpace(q.TenantID) == "" || strings.TrimSpace(q.UserID) == "" {
		return model.EffectivePolicy{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and user_id are required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return model.EffectivePolicy{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return s.resolveEffectivePolicy(ctx, q.TenantID, q.UserID)
}

func (s *CoreService) RegisterDevice(ctx context.Context, actor model.ActorPrincipal, in model.RegisterDeviceRequest) (model.Device, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.DeviceFingerprint) == "" || strings.TrimSpace(in.Platform) == "" {
		return model.Device{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id, user_id, device_fingerprint and platform are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Device{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	effective, err := s.resolveEffectivePolicy(ctx, in.TenantID, in.UserID)
	if err != nil {
		return model.Device{}, err
	}
	if err := s.enforceQuota(effective); err != nil {
		return model.Device{}, err
	}

	existing, err := s.store.GetDeviceByFingerprint(ctx, in.TenantID, in.DeviceFingerprint)
	if err == nil {
		if existing.UserID != in.UserID {
			return model.Device{}, &AppError{Status: 409, Code: "device_already_bound", Message: "device fingerprint already bound to another user"}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.Device{}, &AppError{Status: 500, Code: "device_lookup_failed", Message: "failed to lookup device", Err: err}
	}

	if errors.Is(err, sql.ErrNoRows) && effective.DeviceLimit != nil {
		activeCount, countErr := s.store.CountActiveDevicesForUser(ctx, in.TenantID, in.UserID)
		if countErr != nil {
			return model.Device{}, &AppError{Status: 500, Code: "device_count_failed", Message: "failed to evaluate device limits", Err: countErr}
		}
		if activeCount >= *effective.DeviceLimit {
			return model.Device{}, &AppError{Status: 409, Code: "device_limit_exceeded", Message: "device limit exceeded for user"}
		}
	}

	item, regErr := s.store.RegisterDevice(ctx, in)
	if regErr != nil {
		return model.Device{}, mapStoreError("device_register_failed", regErr)
	}
	return item, nil
}

func (s *CoreService) ListNodes(ctx context.Context, actor model.ActorPrincipal, q model.ListNodesQuery) ([]model.Node, error) {
	if !actor.IsPlatformAdmin() {
		if strings.TrimSpace(q.TenantID) == "" {
			q.TenantID = actor.TenantID
		}
	}
	if strings.TrimSpace(q.TenantID) == "" {
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	items, err := s.store.ListNodes(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "node_list_failed", Message: "failed to list nodes", Err: err}
	}
	for i := range items {
		items[i].Status = s.deriveNodeStatus(items[i], time.Now().UTC())
	}
	return items, nil
}

func (s *CoreService) RegisterNode(ctx context.Context, actor model.ActorPrincipal, in model.RegisterNodeRequest) (model.NodeRegistrationResult, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.Region) == "" || strings.TrimSpace(in.Hostname) == "" || strings.TrimSpace(in.NodeKeyID) == "" || strings.TrimSpace(in.NodePublicKey) == "" || strings.TrimSpace(in.AgentVersion) == "" {
		return model.NodeRegistrationResult{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id, region, hostname, node_key_id, node_public_key and agent_version are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeRegistrationResult{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if err := s.validateNodeContractVersion(in.ContractVersion); err != nil {
		return model.NodeRegistrationResult{}, err
	}
	if err := s.validateRequiredCapabilities(in.Capabilities); err != nil {
		return model.NodeRegistrationResult{}, err
	}
	if err := s.validateSignedAt(in.SignedAt); err != nil {
		return model.NodeRegistrationResult{}, err
	}
	if err := s.verifyRegisterSignature(in); err != nil {
		return model.NodeRegistrationResult{}, err
	}

	fingerprint := hashString(in.NodePublicKey)
	node, err := s.store.UpsertNodeRegistration(ctx, in, fingerprint)
	if err != nil {
		return model.NodeRegistrationResult{}, mapStoreError("node_register_failed", err)
	}
	if node.Status == "revoked" {
		return model.NodeRegistrationResult{}, &AppError{Status: 403, Code: "node_revoked", Message: "node is revoked"}
	}
	node.Status = s.deriveNodeStatus(node, time.Now().UTC())
	provisioning := s.buildProvisioningContract(node)

	if hbErr := s.store.InsertNodeHeartbeatEvent(ctx, node.ID, node.TenantID, "registered", map[string]any{"node_key_id": node.NodeKeyID, "agent_version": node.AgentVersion}); hbErr != nil {
		return model.NodeRegistrationResult{}, mapStoreError("node_register_failed", hbErr)
	}

	return model.NodeRegistrationResult{Node: node, Provisioning: provisioning}, nil
}

func (s *CoreService) NodeHeartbeat(ctx context.Context, actor model.ActorPrincipal, in model.NodeHeartbeatRequest) (model.Node, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.NodeID) == "" || strings.TrimSpace(in.NodeKeyID) == "" || strings.TrimSpace(in.AgentVersion) == "" {
		return model.Node{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id, node_id, node_key_id and agent_version are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if err := s.validateNodeContractVersion(in.ContractVersion); err != nil {
		return model.Node{}, err
	}
	if err := s.validateSignedAt(in.SignedAt); err != nil {
		return model.Node{}, err
	}

	nodeByID, err := s.store.GetNodeByID(ctx, in.TenantID, in.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.Node{}, &AppError{Status: 500, Code: "node_heartbeat_failed", Message: "failed to load node", Err: err}
	}
	if nodeByID.NodeKeyID != in.NodeKeyID {
		return model.Node{}, &AppError{Status: 403, Code: "invalid_node_identity", Message: "node identity mismatch"}
	}
	if nodeByID.Status == "revoked" {
		return model.Node{}, &AppError{Status: 403, Code: "node_revoked", Message: "node is revoked"}
	}
	if err := s.verifyHeartbeatSignature(in, nodeByID.NodePublicKey); err != nil {
		return model.Node{}, err
	}

	node, err := s.store.TouchNodeHeartbeat(ctx, in)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.Node{}, mapStoreError("node_heartbeat_failed", err)
	}
	node.Status = s.deriveNodeStatus(node, time.Now().UTC())
	if hbErr := s.store.InsertNodeHeartbeatEvent(ctx, node.ID, node.TenantID, "heartbeat", map[string]any{"status": node.Status}); hbErr != nil {
		return model.Node{}, mapStoreError("node_heartbeat_failed", hbErr)
	}
	return node, nil
}

func (s *CoreService) GetNodeProvisioning(ctx context.Context, actor model.ActorPrincipal, q model.GetNodeProvisioningQuery) (model.NodeProvisioningContract, error) {
	if strings.TrimSpace(q.TenantID) == "" {
		return model.NodeProvisioningContract{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id is required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return model.NodeProvisioningContract{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if strings.TrimSpace(q.NodeID) == "" && strings.TrimSpace(q.NodeKeyID) == "" {
		return model.NodeProvisioningContract{}, &AppError{Status: 400, Code: "validation_error", Message: "node_id or node_key_id is required"}
	}

	var (
		node model.Node
		err  error
	)
	if strings.TrimSpace(q.NodeID) != "" {
		node, err = s.store.GetNodeByID(ctx, q.TenantID, q.NodeID)
	} else {
		node, err = s.store.GetNodeByKey(ctx, q.TenantID, q.NodeKeyID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeProvisioningContract{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.NodeProvisioningContract{}, &AppError{Status: 500, Code: "node_provisioning_failed", Message: "failed to get node provisioning", Err: err}
	}
	node.Status = s.deriveNodeStatus(node, time.Now().UTC())
	if node.Status == "revoked" {
		return model.NodeProvisioningContract{}, &AppError{Status: 403, Code: "node_revoked", Message: "node is revoked"}
	}
	return s.buildProvisioningContract(node), nil
}

func (s *CoreService) resolveEffectivePolicy(ctx context.Context, tenantID string, userID string) (model.EffectivePolicy, error) {
	effective, err := s.store.GetEffectivePolicy(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.EffectivePolicy{}, &AppError{Status: 400, Code: "invalid_reference", Message: "user does not belong to tenant", Err: err}
		}
		return model.EffectivePolicy{}, &AppError{Status: 500, Code: "effective_policy_failed", Message: "failed to resolve effective policy", Err: err}
	}

	usage, err := s.store.GetTenantUsageTotal(ctx, tenantID)
	if err != nil {
		return model.EffectivePolicy{}, &AppError{Status: 500, Code: "usage_lookup_failed", Message: "failed to resolve tenant usage", Err: err}
	}
	effective.UsageBytes = usage
	if effective.TrafficQuotaBytes != nil {
		effective.QuotaExceeded = usage >= *effective.TrafficQuotaBytes
	}
	return effective, nil
}

func (s *CoreService) enforceQuota(effective model.EffectivePolicy) error {
	if effective.TrafficQuotaBytes == nil {
		return nil
	}
	if effective.UsageBytes >= *effective.TrafficQuotaBytes {
		return &AppError{Status: 409, Code: "quota_exceeded", Message: "traffic quota exceeded for tenant"}
	}
	return nil
}

func (s *CoreService) validateNodeContractVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return &AppError{Status: 400, Code: "validation_error", Message: "contract_version is required"}
	}
	if strings.TrimSpace(version) != s.opts.NodeContractVersion {
		return &AppError{Status: 400, Code: "unsupported_contract", Message: "unsupported contract version"}
	}
	return nil
}

func (s *CoreService) validateRequiredCapabilities(capabilities []string) error {
	if len(capabilities) == 0 {
		return &AppError{Status: 400, Code: "validation_error", Message: "capabilities are required"}
	}
	available := make(map[string]struct{}, len(capabilities))
	for _, c := range capabilities {
		c = strings.TrimSpace(c)
		if c != "" {
			available[c] = struct{}{}
		}
	}
	for _, required := range s.opts.NodeRequiredCapabilities {
		if _, ok := available[required]; !ok {
			return &AppError{Status: 400, Code: "missing_capability", Message: "missing required capability: " + required}
		}
	}
	return nil
}

func (s *CoreService) validateSignedAt(signedAt int64) error {
	if signedAt <= 0 {
		return &AppError{Status: 400, Code: "validation_error", Message: "signed_at is required"}
	}
	t := time.Unix(signedAt, 0).UTC()
	now := time.Now().UTC()
	if now.Sub(t) > s.opts.NodeSignatureMaxSkew || t.Sub(now) > s.opts.NodeSignatureMaxSkew {
		return &AppError{Status: 401, Code: "invalid_signature", Message: "signature timestamp is outside allowed skew"}
	}
	return nil
}

func (s *CoreService) verifyRegisterSignature(in model.RegisterNodeRequest) error {
	caps := make([]string, 0, len(in.Capabilities))
	for _, c := range in.Capabilities {
		c = strings.TrimSpace(c)
		if c != "" {
			caps = append(caps, c)
		}
	}
	sort.Strings(caps)
	payload := strings.Join([]string{
		"register",
		in.TenantID,
		in.Region,
		in.Hostname,
		in.NodeKeyID,
		in.NodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		strings.Join(caps, ","),
		strconv.FormatInt(in.SignedAt, 10),
	}, "\n")
	if !secureCompareSignature(in.Signature, signPayload(s.opts.NodeSigningSecret, payload)) {
		return &AppError{Status: 401, Code: "invalid_signature", Message: "invalid node registration signature"}
	}
	return nil
}

func (s *CoreService) verifyHeartbeatSignature(in model.NodeHeartbeatRequest, nodePublicKey string) error {
	payload := strings.Join([]string{
		"heartbeat",
		in.TenantID,
		in.NodeID,
		in.NodeKeyID,
		nodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		strconv.FormatInt(in.SignedAt, 10),
	}, "\n")
	if !secureCompareSignature(in.Signature, signPayload(s.opts.NodeSigningSecret, payload)) {
		return &AppError{Status: 401, Code: "invalid_signature", Message: "invalid node heartbeat signature"}
	}
	return nil
}

func (s *CoreService) deriveNodeStatus(node model.Node, now time.Time) string {
	if node.Status == "revoked" {
		return "revoked"
	}
	if node.LastHeartbeatAt == nil {
		return "pending"
	}
	age := now.Sub(node.LastHeartbeatAt.UTC())
	switch {
	case age <= s.opts.NodeStaleAfter:
		return "active"
	case age <= s.opts.NodeOfflineAfter:
		return "stale"
	default:
		return "offline"
	}
}

func (s *CoreService) buildProvisioningContract(node model.Node) model.NodeProvisioningContract {
	return model.NodeProvisioningContract{
		TenantID:                 node.TenantID,
		NodeID:                   node.ID,
		NodeKeyID:                node.NodeKeyID,
		ContractVersion:          s.opts.NodeContractVersion,
		HeartbeatIntervalSeconds: int(s.opts.NodeHeartbeatInterval.Seconds()),
		StaleAfterSeconds:        int(s.opts.NodeStaleAfter.Seconds()),
		OfflineAfterSeconds:      int(s.opts.NodeOfflineAfter.Seconds()),
		RequiredCapabilities:     append([]string(nil), s.opts.NodeRequiredCapabilities...),
	}
}

func validatePolicyValues(trafficQuota *int64, deviceLimit *int, ttlSeconds *int) error {
	if trafficQuota != nil && *trafficQuota < 0 {
		return &AppError{Status: 400, Code: "validation_error", Message: "traffic_quota_bytes must be >= 0"}
	}
	if deviceLimit != nil && *deviceLimit <= 0 {
		return &AppError{Status: 400, Code: "validation_error", Message: "device_limit must be > 0"}
	}
	if ttlSeconds != nil && *ttlSeconds <= 0 {
		return &AppError{Status: 400, Code: "validation_error", Message: "ttl must be > 0"}
	}
	return nil
}

func mapStoreError(defaultCode string, err error) error {
	var dbErr *store.DBError
	if errors.As(err, &dbErr) {
		switch dbErr.Kind {
		case store.ErrorKindConflict:
			return &AppError{Status: 409, Code: "conflict", Message: "resource already exists", Err: err}
		case store.ErrorKindForeignKey:
			return &AppError{Status: 400, Code: "invalid_reference", Message: "related entity does not exist", Err: err}
		case store.ErrorKindValidation:
			return &AppError{Status: 400, Code: "validation_error", Message: dbErr.Message, Err: err}
		}
	}
	return &AppError{Status: 500, Code: defaultCode, Message: "internal error", Err: err}
}

func ToResponse(err error, fallbackCode string, fallbackMsg string) (int, string, string) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Code, appErr.Message
	}
	return 500, fallbackCode, fallbackMsg
}

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func hashString(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func signPayload(secret, payload string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func secureCompareSignature(got, want string) bool {
	got = strings.ToLower(strings.TrimSpace(got))
	want = strings.ToLower(strings.TrimSpace(want))
	if len(got) == 0 || len(got) != len(want) {
		return false
	}
	return hmac.Equal([]byte(got), []byte(want))
}
