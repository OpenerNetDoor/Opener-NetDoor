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

func (f fakeService) BlockUser(_ context.Context, actor model.ActorPrincipal, in model.UserLifecycleRequest) (model.User, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.User{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.User{ID: in.UserID, TenantID: in.TenantID, Email: "u@example.com", Status: "blocked", CreatedAt: time.Now()}, nil
}

func (f fakeService) UnblockUser(_ context.Context, actor model.ActorPrincipal, in model.UserLifecycleRequest) (model.User, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.User{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.User{ID: in.UserID, TenantID: in.TenantID, Email: "u@example.com", Status: "active", CreatedAt: time.Now()}, nil
}

func (f fakeService) DeleteUser(_ context.Context, actor model.ActorPrincipal, in model.UserLifecycleRequest) error {
	if !actor.CanAccessTenant(in.TenantID) {
		return &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if strings.TrimSpace(in.UserID) == "" {
		return &service.AppError{Status: 400, Code: "validation_error", Message: "user_id is required"}
	}
	return nil
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

func (f fakeService) CreateNode(_ context.Context, actor model.ActorPrincipal, in model.CreateNodeRequest) (model.Node, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Node{ID: "n-created", TenantID: in.TenantID, Region: in.Region, Hostname: in.Hostname, NodeKeyID: "node-key-created", Status: "pending", CreatedAt: time.Now()}, nil
}

func (f fakeService) GetNode(_ context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.Node, error) {
	if !actor.CanAccessTenant(tenantID) {
		return model.Node{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Node{ID: nodeID, TenantID: tenantID, Region: "eu", Hostname: "server-1", NodeKeyID: "node-key-1", Status: "active", CreatedAt: time.Now()}, nil
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

func (f fakeService) RevokeNode(_ context.Context, actor model.ActorPrincipal, in model.NodeLifecycleRequest) (model.Node, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Node{ID: in.NodeID, TenantID: in.TenantID, NodeKeyID: "node-key-1", Status: "revoked", CreatedAt: time.Now()}, nil
}

func (f fakeService) ReactivateNode(_ context.Context, actor model.ActorPrincipal, in model.NodeLifecycleRequest) (model.Node, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.Node{ID: in.NodeID, TenantID: in.TenantID, NodeKeyID: "node-key-1", Status: "pending", CreatedAt: time.Now()}, nil
}

func (f fakeService) ListNodeCertificates(_ context.Context, actor model.ActorPrincipal, q model.ListNodeCertificatesQuery) ([]model.NodeCertificate, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.NodeCertificate{{ID: "c1", TenantID: q.TenantID, NodeID: q.NodeID, SerialNumber: "ABC", CertPEM: "pem", CAID: "ca1", Issuer: "issuer", NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), CreatedAt: time.Now()}}, nil
}

func (f fakeService) IssueNodeCertificate(_ context.Context, actor model.ActorPrincipal, in model.RotateNodeCertificateRequest) (model.NodeCertificate, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.NodeCertificate{ID: "c2", TenantID: in.TenantID, NodeID: in.NodeID, SerialNumber: "ISSUE", CertPEM: "pem", CAID: "ca1", Issuer: "issuer", NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), CreatedAt: time.Now()}, nil
}

func (f fakeService) RotateNodeCertificate(_ context.Context, actor model.ActorPrincipal, in model.RotateNodeCertificateRequest) (model.NodeCertificate, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.NodeCertificate{ID: "c3", TenantID: in.TenantID, NodeID: in.NodeID, SerialNumber: "ROTATE", CertPEM: "pem", CAID: "ca1", Issuer: "issuer", NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), CreatedAt: time.Now()}, nil
}

func (f fakeService) RevokeNodeCertificate(_ context.Context, actor model.ActorPrincipal, in model.RevokeNodeCertificateRequest) (model.NodeCertificate, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	now := time.Now()
	return model.NodeCertificate{ID: "c4", TenantID: in.TenantID, NodeID: in.NodeID, SerialNumber: "REVOKE", CertPEM: "pem", CAID: "ca1", Issuer: "issuer", NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), RevokedAt: &now, CreatedAt: now}, nil
}
func (f fakeService) RenewNodeCertificate(_ context.Context, actor model.ActorPrincipal, in model.RenewNodeCertificateRequest) (model.RenewNodeCertificateResult, error) {
	if !actor.CanAccessTenant(in.TenantID) {
		return model.RenewNodeCertificateResult{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.RenewNodeCertificateResult{
		TenantID:              in.TenantID,
		NodeID:                in.NodeID,
		PreviousCertificateID: "c-prev",
		PreviousSerialNumber:  "SERIAL-PREV",
		Certificate:           model.NodeCertificate{ID: "c5", TenantID: in.TenantID, NodeID: in.NodeID, SerialNumber: "RENEW", CertPEM: "pem", CAID: "ca1", IssuerID: "issuer-1", Issuer: "issuer", NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), CreatedAt: time.Now()},
		Renewed:               true,
	}, nil
}

func (f fakeService) ListPKIIssuers(_ context.Context, actor model.ActorPrincipal, q model.ListPKIIssuersQuery) ([]model.PKIIssuer, error) {
	if !actor.IsPlatformAdmin() {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	return []model.PKIIssuer{{ID: "pi-1", IssuerID: "issuer-1", Source: "file", CAID: "ca1", IssuerName: "issuer", CACertPEM: "pem", Status: "active", CreatedAt: time.Now()}}, nil
}

func (f fakeService) CreatePKIIssuer(_ context.Context, actor model.ActorPrincipal, in model.CreatePKIIssuerRequest) (model.PKIIssuer, error) {
	if !actor.IsPlatformAdmin() {
		return model.PKIIssuer{}, &service.AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	if strings.TrimSpace(in.IssuerID) == "" {
		return model.PKIIssuer{}, &service.AppError{Status: 400, Code: "validation_error", Message: "issuer_id is required"}
	}
	return model.PKIIssuer{ID: "pi-2", IssuerID: in.IssuerID, Source: "file", CAID: "ca1", IssuerName: "issuer", CACertPEM: "pem", Status: "pending", CreatedAt: time.Now()}, nil
}

func (f fakeService) ActivatePKIIssuer(_ context.Context, actor model.ActorPrincipal, in model.ActivatePKIIssuerRequest) (model.CARotationResult, error) {
	if !actor.IsPlatformAdmin() {
		return model.CARotationResult{}, &service.AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	if strings.TrimSpace(in.IssuerID) == "" {
		return model.CARotationResult{}, &service.AppError{Status: 400, Code: "validation_error", Message: "issuer_id is required"}
	}
	active := model.PKIIssuer{ID: "pi-2", IssuerID: in.IssuerID, Source: "file", CAID: "ca2", IssuerName: "issuer2", CACertPEM: "pem", Status: "active", CreatedAt: time.Now()}
	previous := model.PKIIssuer{ID: "pi-1", IssuerID: "issuer-1", Source: "file", CAID: "ca1", IssuerName: "issuer1", CACertPEM: "pem", Status: "retired", CreatedAt: time.Now()}
	return model.CARotationResult{ActiveIssuer: active, PreviousIssuer: &previous, RotatedAt: time.Now()}, nil
}

func (f fakeService) GetNodeProvisioning(_ context.Context, actor model.ActorPrincipal, q model.GetNodeProvisioningQuery) (model.NodeProvisioningContract, error) {
	if !actor.CanAccessTenant(q.TenantID) {
		return model.NodeProvisioningContract{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.NodeProvisioningContract{TenantID: q.TenantID, NodeID: "n1", NodeKeyID: "node-key-1", ContractVersion: "2026-03-10.stage5.v1"}, nil
}

func (f fakeService) ListAuditLogs(_ context.Context, actor model.ActorPrincipal, q model.ListAuditLogsQuery) ([]model.AuditLogRecord, error) {
	if q.TenantID != "" && !actor.CanAccessTenant(q.TenantID) {
		return nil, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return []model.AuditLogRecord{{
		ID:        "al1",
		TenantID:  q.TenantID,
		ActorType: "node",
		Action:    "node.heartbeat_accepted",
		CreatedAt: time.Now(),
	}}, nil
}

func (f fakeService) GetOpsSnapshot(_ context.Context, actor model.ActorPrincipal, tenantID string) (model.OpsSnapshot, error) {
	if tenantID != "" && !actor.CanAccessTenant(tenantID) {
		return model.OpsSnapshot{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.OpsSnapshot{
		TenantID:    tenantID,
		GeneratedAt: time.Now(),
		NodeStatus: []model.OpsNodeStatusCount{{
			Status: "active",
			Count:  1,
		}},
		ActiveCertificates:   1,
		ExpiringCertificates: 1,
		TrafficBytes24h:      100,
		ReplayRejected24h:    0,
		InvalidSignature24h:  0,
	}, nil
}

func (f fakeService) GetOpsAnalytics(_ context.Context, actor model.ActorPrincipal, tenantID string) (model.OpsAnalytics, error) {
	if tenantID != "" && !actor.CanAccessTenant(tenantID) {
		return model.OpsAnalytics{}, &service.AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	return model.OpsAnalytics{
		TenantID:        tenantID,
		GeneratedAt:     time.Now(),
		TotalUsers:      10,
		ActiveUsers:     8,
		ActiveKeys:      12,
		OnlineServers:   3,
		TrafficBytes24h: 1024,
	}, nil
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

func TestDeleteUser(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodDelete, "/internal/v1/users?id=u1&tenant_id=tenant-a", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.Users(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestBlockUser(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/users/block", strings.NewReader(`{"tenant_id":"tenant-a","user_id":"u1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.UsersBlock(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestUnblockUser(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/users/unblock", strings.NewReader(`{"tenant_id":"tenant-a","user_id":"u1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.UsersUnblock(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
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

func TestNodeRevoke(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes/revoke", strings.NewReader(`{"tenant_id":"tenant-a","node_id":"n1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.NodeRevoke(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestNodeReactivate(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes/reactivate", strings.NewReader(`{"tenant_id":"tenant-a","node_id":"n1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.NodeReactivate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestNodeCreate(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes", strings.NewReader(`{"tenant_id":"tenant-a","region":"de","hostname":"de-1.example.com","capabilities":["heartbeat.v1"]}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.Nodes(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}
}

func TestNodeDetail(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/nodes/detail?tenant_id=tenant-a&node_id=n1", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.NodeDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}
func TestNodeCertificatesList(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/nodes/certificates?tenant_id=tenant-a&node_id=n1", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.NodeCertificates(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestNodeCertificatesRotate(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes/certificates/rotate", strings.NewReader(`{"tenant_id":"tenant-a","node_id":"n1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.NodeCertificatesRotate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestNodeCertificatesRenew(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/nodes/certificates/renew", strings.NewReader(`{"tenant_id":"tenant-a","node_id":"n1"}`))
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:write")
	rr := httptest.NewRecorder()

	h.NodeCertificatesRenew(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestPKIIssuersPlatformAdminOnly(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/pki/issuers", nil)
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.PKIIssuers(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d body=%s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

func TestPKIIssuersActivate(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/pki/issuers/activate", strings.NewReader(`{"issuer_id":"issuer-2"}`))
	req.Header.Set("X-Actor-Scopes", "admin:write,platform:admin")
	rr := httptest.NewRecorder()

	h.PKIIssuersActivate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestAuditLogsEndpoint(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/audit/logs?tenant_id=tenant-a", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.AuditLogs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestOpsSnapshotEndpoint(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/ops/snapshot?tenant_id=tenant-a", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.OpsSnapshot(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestOpsAnalyticsEndpoint(t *testing.T) {
	h := New(fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/ops/analytics?tenant_id=tenant-a", nil)
	req.Header.Set("X-Actor-Tenant-ID", "tenant-a")
	req.Header.Set("X-Actor-Scopes", "admin:read")
	rr := httptest.NewRecorder()

	h.OpsAnalytics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}
