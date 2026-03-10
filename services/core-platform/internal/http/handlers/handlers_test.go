package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/service"
)

type fakeService struct {
	healthErr error
}

func (f fakeService) Health(context.Context) error {
	return f.healthErr
}

func (f fakeService) ListTenants(context.Context, model.ActorPrincipal, model.ListQuery) ([]model.Tenant, error) {
	return []model.Tenant{{ID: "t1", Name: "Tenant1", Status: "active", CreatedAt: time.Now()}}, nil
}

func (f fakeService) CreateTenant(_ context.Context, actor model.ActorPrincipal, in model.CreateTenantRequest) (model.Tenant, error) {
	if !actor.IsPlatformAdmin() {
		return model.Tenant{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot create tenants"}
	}
	return model.Tenant{ID: "t2", Name: in.Name, Status: "active", CreatedAt: time.Now()}, nil
}

func (f fakeService) ListUsers(_ context.Context, actor model.ActorPrincipal, q model.ListUsersQuery) ([]model.User, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.User{{ID: "u1", TenantID: q.TenantID, Email: "u@example.com", Status: "active", CreatedAt: time.Now()}}, nil
}

func (f fakeService) CreateUser(_ context.Context, actor model.ActorPrincipal, in model.CreateUserRequest) (model.User, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.User{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.User{ID: "u2", TenantID: in.TenantID, Email: in.Email, Status: "active", CreatedAt: time.Now()}, nil
}

func (f fakeService) ListAccessKeys(_ context.Context, actor model.ActorPrincipal, q model.ListAccessKeysQuery) ([]model.AccessKey, error) {
	if q.TenantID != "" && !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.AccessKey{{
		ID:        "k1",
		TenantID:  q.TenantID,
		UserID:    "u1",
		KeyType:   "vless",
		SecretRef: "secret://k1",
		Status:    "active",
		CreatedAt: time.Now(),
	}}, nil
}

func (f fakeService) CreateAccessKey(_ context.Context, actor model.ActorPrincipal, in model.CreateAccessKeyRequest) (model.AccessKey, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.AccessKey{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.AccessKey{
		ID:        "k2",
		TenantID:  in.TenantID,
		UserID:    in.UserID,
		KeyType:   in.KeyType,
		SecretRef: "secret://k2",
		Status:    "active",
		CreatedAt: time.Now(),
	}, nil
}

func (f fakeService) RevokeAccessKey(_ context.Context, actor model.ActorPrincipal, in model.RevokeAccessKeyRequest) (model.AccessKey, error) {
	if in.TenantID != "" && !actor.CanAccessTenant(in.TenantID) {
		return model.AccessKey{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.AccessKey{
		ID:        in.ID,
		TenantID:  in.TenantID,
		UserID:    "u1",
		KeyType:   "vless",
		SecretRef: "secret://k1",
		Status:    "revoked",
		CreatedAt: time.Now(),
	}, nil
}

func (f fakeService) ListTenantPolicies(_ context.Context, actor model.ActorPrincipal, q model.ListTenantPoliciesQuery) ([]model.TenantPolicy, error) {
	if q.TenantID != "" && !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.TenantPolicy{{TenantID: "tenant-a", UpdatedAt: time.Now()}}, nil
}

func (f fakeService) GetTenantPolicy(_ context.Context, actor model.ActorPrincipal, tenantID string) (model.TenantPolicy, error) {
	if !actor.CanAccessTenant(tenantID) {
		return model.TenantPolicy{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.TenantPolicy{TenantID: tenantID, UpdatedAt: time.Now()}, nil
}

func (f fakeService) SetTenantPolicy(_ context.Context, actor model.ActorPrincipal, in model.SetTenantPolicyRequest) (model.TenantPolicy, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.TenantPolicy{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.TenantPolicy{TenantID: in.TenantID, UpdatedAt: time.Now(), TrafficQuotaBytes: in.TrafficQuotaBytes}, nil
}

func (f fakeService) ListUserPolicyOverrides(_ context.Context, actor model.ActorPrincipal, q model.ListUserPolicyOverridesQuery) ([]model.UserPolicyOverride, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.UserPolicyOverride{{TenantID: q.TenantID, UserID: "u1", UpdatedAt: time.Now()}}, nil
}

func (f fakeService) GetUserPolicyOverride(_ context.Context, actor model.ActorPrincipal, tenantID string, userID string) (model.UserPolicyOverride, error) {
	if !actor.CanAccessTenant(tenantID) {
		return model.UserPolicyOverride{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.UserPolicyOverride{TenantID: tenantID, UserID: userID, UpdatedAt: time.Now()}, nil
}

func (f fakeService) SetUserPolicyOverride(_ context.Context, actor model.ActorPrincipal, in model.SetUserPolicyOverrideRequest) (model.UserPolicyOverride, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.UserPolicyOverride{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.UserPolicyOverride{TenantID: in.TenantID, UserID: in.UserID, UpdatedAt: time.Now()}, nil
}

func (f fakeService) GetEffectivePolicy(_ context.Context, actor model.ActorPrincipal, q model.GetEffectivePolicyQuery) (model.EffectivePolicy, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return model.EffectivePolicy{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.EffectivePolicy{TenantID: q.TenantID, UserID: q.UserID, Source: "tenant_default"}, nil
}

func (f fakeService) RegisterDevice(_ context.Context, actor model.ActorPrincipal, in model.RegisterDeviceRequest) (model.Device, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Device{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Device{
		ID:                "d1",
		TenantID:          in.TenantID,
		UserID:            in.UserID,
		DeviceFingerprint: in.DeviceFingerprint,
		Platform:          in.Platform,
		Status:            "active",
		CreatedAt:         time.Now(),
	}, nil
}

func (f fakeService) ListNodes(_ context.Context, actor model.ActorPrincipal, q model.ListNodesQuery) ([]model.Node, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.Node{{ID: "n1", TenantID: q.TenantID, NodeKeyID: "node-key-1", Status: "active", Region: "eu", Hostname: "n1", CreatedAt: time.Now()}}, nil
}

func (f fakeService) RegisterNode(_ context.Context, actor model.ActorPrincipal, in model.RegisterNodeRequest) (model.NodeRegistrationResult, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeRegistrationResult{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.NodeRegistrationResult{
		Node:         model.Node{ID: "n1", TenantID: in.TenantID, NodeKeyID: in.NodeKeyID, Status: "pending", Region: in.Region, Hostname: in.Hostname, CreatedAt: time.Now()},
		Provisioning: model.NodeProvisioningContract{NodeID: "n1", TenantID: in.TenantID, NodeKeyID: in.NodeKeyID, ContractVersion: in.ContractVersion},
	}, nil
}

func (f fakeService) NodeHeartbeat(_ context.Context, actor model.ActorPrincipal, in model.NodeHeartbeatRequest) (model.Node, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Node{ID: in.NodeID, TenantID: in.TenantID, NodeKeyID: in.NodeKeyID, Status: "active", CreatedAt: time.Now()}, nil
}

func (f fakeService) GetNodeProvisioning(_ context.Context, actor model.ActorPrincipal, q model.GetNodeProvisioningQuery) (model.NodeProvisioningContract, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return model.NodeProvisioningContract{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.NodeProvisioningContract{TenantID: q.TenantID, NodeID: "n1", NodeKeyID: "node-key-1", ContractVersion: "2026-03-10.stage5.v1"}, nil
}

func TestCreateTenantForbiddenForTenantScopedActor(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/tenants", strings.NewReader(`{"name":"Acme"}`))
	req.Header.Set("X-Actor-Scopes", "admin:write")
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	rr := httptest.NewRecorder()

	h.Tenants(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestCreateTenantAllowedForPlatformAdmin(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/tenants", strings.NewReader(`{"name":"Acme"}`))
	req.Header.Set("X-Actor-Scopes", "admin:write,platform:admin")
	rr := httptest.NewRecorder()

	h.Tenants(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestListUsersDenyTenantIsolation(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/users?tenant_id=tenant-b", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.Users(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestReadyUnavailableWhenHealthFails(t *testing.T) {
	h := New(fakeService{healthErr: &service.AppError{Status: 503, Code: "db_unavailable", Message: "database is unavailable"}})
	req := httptest.NewRequest(http.MethodGet, "/internal/ready", nil)
	rr := httptest.NewRecorder()

	h.Ready(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestAccessKeysRevoke(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodDelete, "/internal/v1/access-keys?id=k1&tenant_id=tenant-a", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.AccessKeys(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestReadyUnavailableWhenHealthFailsByError(t *testing.T) {
	h := New(fakeService{healthErr: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/internal/ready", nil)
	rr := httptest.NewRecorder()

	h.Ready(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestSetTenantPolicy(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPut, "/internal/v1/policies/tenants", strings.NewReader(`{"tenant_id":"tenant-a","traffic_quota_bytes":1000}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.TenantPolicies(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestRegisterDevice(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/devices/register", strings.NewReader(`{"tenant_id":"tenant-a","user_id":"u1","device_fingerprint":"fp1","platform":"ios"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.Devices(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}
}

func TestNodeRegister(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes/register", strings.NewReader(`{"tenant_id":"tenant-a","region":"eu","hostname":"node-1","node_key_id":"nk1","node_public_key":"pk","contract_version":"2026-03-10.stage5.v1","agent_version":"1.0.0","capabilities":["heartbeat.v1","provisioning.v1"],"signed_at":1700000000,"signature":"sig"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.NodeRegister(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}
}
