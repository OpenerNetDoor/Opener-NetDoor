//go:build integration

package service

import (
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestCoreService_AuditLogsAndOpsSnapshot(t *testing.T) {
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
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage9")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	user, err := svc.CreateUser(t.Context(), platform, model.CreateUserRequest{TenantID: tenant.ID, Email: "stage9-user@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	node, err := s.UpsertNodeRegistration(t.Context(), model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "svc-stage9-node",
		NodeKeyID:       "svc-stage9-node-key",
		NodePublicKey:   "svc-stage9-node-pub",
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "svc-stage9-register-nonce",
		SignedAt:        time.Now().UTC().Unix(),
	}, "svc-stage9-fingerprint")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	now := time.Now().UTC()
	_, err = s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       node.ID,
		SerialNumber: "SVC-STAGE9-CERT-1",
		CertPEM:      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		CAID:         "svc-stage9-ca",

		Issuer:    "svc-stage9",
		NotBefore: now.Add(-1 * time.Hour),
		NotAfter:  now.Add(12 * time.Hour),
	})
	if err != nil {
		t.Fatalf("issue node certificate: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 300, 200)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour)); err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	if err := s.InsertAuditLog(t.Context(), model.AuditLogEvent{TenantID: tenant.ID, ActorType: "node", ActorSub: "n1", Action: "node.replay_rejected", TargetType: "node", TargetID: node.ID, OccurredAt: now}); err != nil {
		t.Fatalf("insert replay audit: %v", err)
	}
	if err := s.InsertAuditLog(t.Context(), model.AuditLogEvent{TenantID: tenant.ID, ActorType: "node", ActorSub: "n1", Action: "node.invalid_signature", TargetType: "node", TargetID: node.ID, OccurredAt: now}); err != nil {
		t.Fatalf("insert invalid signature audit: %v", err)
	}

	tenantActor := testutil.TenantActor(tenant.ID)
	logs, err := svc.ListAuditLogs(t.Context(), tenantActor, model.ListAuditLogsQuery{ListQuery: model.ListQuery{Limit: 20}})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", len(logs))
	}

	snapshot, err := svc.GetOpsSnapshot(t.Context(), tenantActor, "")
	if err != nil {
		t.Fatalf("get ops snapshot: %v", err)
	}
	if snapshot.TenantID != tenant.ID {
		t.Fatalf("expected tenant scoped snapshot tenant_id=%s, got %s", tenant.ID, snapshot.TenantID)
	}
	if snapshot.ActiveCertificates < 1 {
		t.Fatalf("expected active certificates >= 1, got %d", snapshot.ActiveCertificates)
	}
	if snapshot.ReplayRejected24h < 1 {
		t.Fatalf("expected replay_rejected_24h >= 1, got %d", snapshot.ReplayRejected24h)
	}
	if snapshot.InvalidSignature24h < 1 {
		t.Fatalf("expected invalid_signature_24h >= 1, got %d", snapshot.InvalidSignature24h)
	}
	if snapshot.TrafficBytes24h < 500 {
		t.Fatalf("expected traffic bytes >= 500, got %d", snapshot.TrafficBytes24h)
	}
}
