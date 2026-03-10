//go:build integration

package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

const (
	integrationNodeSigningSecret   = "opener-netdoor-stage5-dev-signing-secret"
	integrationNodeContractVersion = "2026-03-10.stage5.v1"
)

func TestCoreService_CreateTenantConflict(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s)
	actor := testutil.PlatformAdminActor()
	tenantName := testutil.UniqueName("tenant")

	if _, err := svc.CreateTenant(t.Context(), actor, model.CreateTenantRequest{Name: tenantName}); err != nil {
		t.Fatalf("create tenant first time: %v", err)
	}
	_, err = svc.CreateTenant(t.Context(), actor, model.CreateTenantRequest{Name: tenantName})
	if err == nil {
		t.Fatal("expected conflict error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Status != 409 || appErr.Code != "conflict" {
		t.Fatalf("expected 409/conflict, got %d/%s", appErr.Status, appErr.Code)
	}
}

func TestCoreService_CreateUserInvalidReference(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s)
	actor := testutil.PlatformAdminActor()

	_, err = svc.CreateUser(t.Context(), actor, model.CreateUserRequest{
		TenantID: "00000000-0000-0000-0000-000000000000",
		Email:    "missing-tenant@example.com",
		Note:     "expect invalid reference",
	})
	if err == nil {
		t.Fatal("expected invalid_reference error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Status != 400 || appErr.Code != "invalid_reference" {
		t.Fatalf("expected 400/invalid_reference, got %d/%s", appErr.Status, appErr.Code)
	}
}

func TestCoreService_TenantIsolationDeny(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s)
	platform := testutil.PlatformAdminActor()
	tenantA, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-a")})
	if err != nil {
		t.Fatalf("create tenant a: %v", err)
	}
	tenantB, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-b")})
	if err != nil {
		t.Fatalf("create tenant b: %v", err)
	}

	tenantActor := testutil.TenantActor(tenantA.ID)
	_, err = svc.ListUsers(t.Context(), tenantActor, model.ListUsersQuery{TenantID: tenantB.ID, ListQuery: model.ListQuery{Limit: 10}})
	if err == nil {
		t.Fatal("expected forbidden error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Status != 403 || appErr.Code != "forbidden" {
		t.Fatalf("expected 403/forbidden, got %d/%s", appErr.Status, appErr.Code)
	}
}

func TestCoreService_AccessKeyLifecycle(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s)
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := svc.CreateUser(t.Context(), platform, model.CreateUserRequest{TenantID: tenant.ID, Email: "svc-ak@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := svc.CreateAccessKey(t.Context(), platform, model.CreateAccessKeyRequest{TenantID: tenant.ID, UserID: user.ID, KeyType: "vless"})
	if err != nil {
		t.Fatalf("create access key: %v", err)
	}

	items, err := svc.ListAccessKeys(t.Context(), platform, model.ListAccessKeysQuery{TenantID: tenant.ID, UserID: user.ID, ListQuery: model.ListQuery{Limit: 10}})
	if err != nil {
		t.Fatalf("list access keys: %v", err)
	}
	if len(items) == 0 || items[0].ID != created.ID {
		t.Fatalf("expected created key in list, got %+v", items)
	}

	revoked, err := svc.RevokeAccessKey(t.Context(), platform, model.RevokeAccessKeyRequest{ID: created.ID, TenantID: tenant.ID})
	if err != nil {
		t.Fatalf("revoke access key: %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", revoked.Status)
	}
}

func TestCoreService_PolicyEnforcement(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s)
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := svc.CreateUser(t.Context(), platform, model.CreateUserRequest{TenantID: tenant.ID, Email: "svc-policy@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	quota := int64(1000)
	deviceLimit := 1
	ttl := 600
	if _, err := svc.SetTenantPolicy(t.Context(), platform, model.SetTenantPolicyRequest{
		TenantID:             tenant.ID,
		TrafficQuotaBytes:    &quota,
		DeviceLimit:          &deviceLimit,
		DefaultKeyTTLSeconds: &ttl,
	}); err != nil {
		t.Fatalf("set tenant policy: %v", err)
	}

	if _, err := svc.RegisterDevice(t.Context(), platform, model.RegisterDeviceRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		DeviceFingerprint: "fp-1",
		Platform:          "android",
	}); err != nil {
		t.Fatalf("register first device: %v", err)
	}

	_, err = svc.RegisterDevice(t.Context(), platform, model.RegisterDeviceRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		DeviceFingerprint: "fp-2",
		Platform:          "ios",
	})
	if err == nil {
		t.Fatal("expected device_limit_exceeded error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError for device limit, got %T (%v)", err, err)
	}
	if appErr.Code != "device_limit_exceeded" {
		t.Fatalf("expected device_limit_exceeded, got %s", appErr.Code)
	}

	_, err = db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 700, 600)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour))
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	_, err = svc.CreateAccessKey(t.Context(), platform, model.CreateAccessKeyRequest{TenantID: tenant.ID, UserID: user.ID, KeyType: "vless"})
	if err == nil {
		t.Fatal("expected quota_exceeded error")
	}
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError for quota, got %T (%v)", err, err)
	}
	if appErr.Code != "quota_exceeded" {
		t.Fatalf("expected quota_exceeded, got %s", appErr.Code)
	}
}

func TestCoreService_NodeRegistrationAndHeartbeat(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s, Options{NodeSigningSecret: integrationNodeSigningSecret, NodeContractVersion: integrationNodeContractVersion})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	register := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-svc-1",
		NodeKeyID:       "node-key-svc-1",
		NodePublicKey:   "pubkey-svc-1",
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		SignedAt:        time.Now().UTC().Unix(),
	}
	register.Signature = signRegister(register)

	registered, err := svc.RegisterNode(t.Context(), platform, register)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}
	if registered.Node.ID == "" {
		t.Fatal("expected node id")
	}

	heartbeat := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       register.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	heartbeat.Signature = signHeartbeat(heartbeat, register.NodePublicKey)

	node, err := svc.NodeHeartbeat(t.Context(), platform, heartbeat)
	if err != nil {
		t.Fatalf("node heartbeat: %v", err)
	}
	if node.Status != "active" {
		t.Fatalf("expected active status after heartbeat, got %s", node.Status)
	}

	nodes, err := svc.ListNodes(t.Context(), platform, model.ListNodesQuery{TenantID: tenant.ID, ListQuery: model.ListQuery{Limit: 10}})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	provisioning, err := svc.GetNodeProvisioning(t.Context(), platform, model.GetNodeProvisioningQuery{TenantID: tenant.ID, NodeID: registered.Node.ID})
	if err != nil {
		t.Fatalf("get node provisioning: %v", err)
	}
	if provisioning.ContractVersion != integrationNodeContractVersion {
		t.Fatalf("unexpected contract version: %s", provisioning.ContractVersion)
	}
}

func signRegister(in model.RegisterNodeRequest) string {
	caps := append([]string(nil), in.Capabilities...)
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
	return sign(payload)
}

func signHeartbeat(in model.NodeHeartbeatRequest, nodePublicKey string) string {
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
	return sign(payload)
}

func sign(payload string) string {
	h := hmac.New(sha256.New, []byte(integrationNodeSigningSecret))
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
