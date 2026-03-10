//go:build integration

package store

import (
	"errors"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestMigrationsApplied(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)

	var tablesCount int
	err := db.QueryRow(`
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_name IN ('tenants','users','admins','nodes','devices','access_keys','traffic_usage_hourly','audit_logs','tenant_policies','user_policy_overrides','node_heartbeats')
	`).Scan(&tablesCount)
	if err != nil {
		t.Fatalf("check tables: %v", err)
	}
	if tablesCount != 11 {
		t.Fatalf("expected 11 tables, got %d", tablesCount)
	}

	var indexCount int
	err = db.QueryRow(`
		SELECT count(*)
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname IN ('uq_tenants_name','uq_users_tenant_email','idx_user_policy_overrides_tenant_updated_at','uq_nodes_tenant_node_key','idx_node_heartbeats_node_received')
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("check indexes: %v", err)
	}
	if indexCount != 5 {
		t.Fatalf("expected 5 hardening indexes, got %d", indexCount)
	}
}

func TestSQLStore_CreateTenantUniqueViolation(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	name := testutil.UniqueName("tenant")
	if _, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: name}); err != nil {
		t.Fatalf("create tenant first time: %v", err)
	}
	_, err = s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: name})
	if err == nil {
		t.Fatal("expected unique violation error")
	}

	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T (%v)", err, err)
	}
	if dbErr.Kind != ErrorKindConflict {
		t.Fatalf("expected conflict kind, got %s", dbErr.Kind)
	}
}

func TestSQLStore_CreateUserForeignKeyViolation(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	_, err = s.CreateUser(t.Context(), model.CreateUserRequest{
		TenantID: "00000000-0000-0000-0000-000000000000",
		Email:    "fk@example.com",
		Note:     "expect fk",
	})
	if err == nil {
		t.Fatal("expected foreign key violation")
	}

	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T (%v)", err, err)
	}
	if dbErr.Kind != ErrorKindForeignKey {
		t.Fatalf("expected foreign_key kind, got %s", dbErr.Kind)
	}
}

func TestSQLStore_AccessKeyLifecycle(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := s.CreateUser(t.Context(), model.CreateUserRequest{TenantID: tenant.ID, Email: "u1@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	key, err := s.CreateAccessKey(t.Context(), model.CreateAccessKeyRequest{TenantID: tenant.ID, UserID: user.ID, KeyType: "vless", SecretRef: "secret://ak1"})
	if err != nil {
		t.Fatalf("create access key: %v", err)
	}

	items, err := s.ListAccessKeys(t.Context(), model.ListAccessKeysQuery{TenantID: tenant.ID, UserID: user.ID, ListQuery: model.ListQuery{Limit: 10}})
	if err != nil {
		t.Fatalf("list access keys: %v", err)
	}
	if len(items) != 1 || items[0].ID != key.ID {
		t.Fatalf("expected listed key %s, got %+v", key.ID, items)
	}

	revoked, err := s.RevokeAccessKey(t.Context(), key.ID, tenant.ID)
	if err != nil {
		t.Fatalf("revoke access key: %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", revoked.Status)
	}
}

func TestSQLStore_PolicyLifecycleAndEffectivePolicy(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	actor := testutil.PlatformAdminActor()
	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := s.CreateUser(t.Context(), model.CreateUserRequest{TenantID: tenant.ID, Email: "policy-u@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	quota := int64(1024)
	devLimit := 2
	ttlDefault := 3600
	tp, err := s.UpsertTenantPolicy(t.Context(), actor, model.SetTenantPolicyRequest{
		TenantID:             tenant.ID,
		TrafficQuotaBytes:    &quota,
		DeviceLimit:          &devLimit,
		DefaultKeyTTLSeconds: &ttlDefault,
	})
	if err != nil {
		t.Fatalf("upsert tenant policy: %v", err)
	}
	if tp.TenantID != tenant.ID {
		t.Fatalf("unexpected tenant policy tenant_id: %s", tp.TenantID)
	}

	userQuota := int64(2048)
	uo, err := s.UpsertUserPolicyOverride(t.Context(), actor, model.SetUserPolicyOverrideRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		TrafficQuotaBytes: &userQuota,
	})
	if err != nil {
		t.Fatalf("upsert user policy override: %v", err)
	}
	if uo.UserID != user.ID {
		t.Fatalf("unexpected user policy user_id: %s", uo.UserID)
	}

	effective, err := s.GetEffectivePolicy(t.Context(), tenant.ID, user.ID)
	if err != nil {
		t.Fatalf("get effective policy: %v", err)
	}
	if effective.TrafficQuotaBytes == nil || *effective.TrafficQuotaBytes != userQuota {
		t.Fatalf("expected user override quota %d, got %+v", userQuota, effective.TrafficQuotaBytes)
	}
	if effective.DeviceLimit == nil || *effective.DeviceLimit != devLimit {
		t.Fatalf("expected inherited device limit %d, got %+v", devLimit, effective.DeviceLimit)
	}
	if effective.KeyTTLSeconds == nil || *effective.KeyTTLSeconds != ttlDefault {
		t.Fatalf("expected inherited ttl %d, got %+v", ttlDefault, effective.KeyTTLSeconds)
	}
}

func TestSQLStore_RegisterDeviceAndUsage(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := s.CreateUser(t.Context(), model.CreateUserRequest{TenantID: tenant.ID, Email: "device-u@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	device, err := s.RegisterDevice(t.Context(), model.RegisterDeviceRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		DeviceFingerprint: "fp-1",
		Platform:          "ios",
	})
	if err != nil {
		t.Fatalf("register device: %v", err)
	}
	if device.Status != "active" {
		t.Fatalf("expected active device, got %s", device.Status)
	}

	count, err := s.CountActiveDevicesForUser(t.Context(), tenant.ID, user.ID)
	if err != nil {
		t.Fatalf("count active devices: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 device, got %d", count)
	}

	_, err = db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 100, 200)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour))
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	total, err := s.GetTenantUsageTotal(t.Context(), tenant.ID)
	if err != nil {
		t.Fatalf("get tenant usage total: %v", err)
	}
	if total != 300 {
		t.Fatalf("expected usage total 300, got %d", total)
	}
}

func TestSQLStore_NodeRegistrationHeartbeatLifecycle(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	node, err := s.UpsertNodeRegistration(t.Context(), model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu",
		Hostname:        "node-1",
		NodeKeyID:       "nk-1",
		NodePublicKey:   "pub-1",
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
	}, "fp")
	if err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	if node.ID == "" {
		t.Fatal("expected node id")
	}

	touched, err := s.TouchNodeHeartbeat(t.Context(), model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          node.ID,
		NodeKeyID:       "nk-1",
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.1",
	})
	if err != nil {
		t.Fatalf("touch heartbeat: %v", err)
	}
	if touched.Status != "active" {
		t.Fatalf("expected active status, got %s", touched.Status)
	}

	nodes, err := s.ListNodes(t.Context(), model.ListNodesQuery{TenantID: tenant.ID, ListQuery: model.ListQuery{Limit: 10}})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	if err := s.InsertNodeHeartbeatEvent(t.Context(), node.ID, tenant.ID, "heartbeat", map[string]any{"ok": true}); err != nil {
		t.Fatalf("insert node heartbeat event: %v", err)
	}
}
