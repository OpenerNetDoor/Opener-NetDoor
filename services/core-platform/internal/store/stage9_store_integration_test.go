//go:build integration

package store

import (
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestSQLStore_AuditLogsAndOpsCounters(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, node := createNodeForCertificateTests(t, s)
	user, err := s.CreateUser(t.Context(), model.CreateUserRequest{TenantID: tenant.ID, Email: "store-stage9-user@example.com"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Now().UTC()
	if err := s.InsertAuditLog(t.Context(), model.AuditLogEvent{TenantID: tenant.ID, ActorType: "node", Action: "node.replay_rejected", TargetType: "node", TargetID: node.ID, OccurredAt: now}); err != nil {
		t.Fatalf("insert audit replay: %v", err)
	}
	if err := s.InsertAuditLog(t.Context(), model.AuditLogEvent{TenantID: tenant.ID, ActorType: "node", Action: "node.invalid_signature", TargetType: "node", TargetID: node.ID, OccurredAt: now}); err != nil {
		t.Fatalf("insert audit invalid signature: %v", err)
	}

	logs, err := s.ListAuditLogs(t.Context(), model.ListAuditLogsQuery{
		ListQuery: model.ListQuery{Limit: 10, Offset: 0},
		TenantID:  tenant.ID,
		Action:    "node.replay_rejected",
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 filtered audit log, got %d", len(logs))
	}
	if logs[0].Action != "node.replay_rejected" {
		t.Fatalf("expected node.replay_rejected action, got %s", logs[0].Action)
	}

	_, err = s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       node.ID,
		SerialNumber: "STORE-STAGE9-CERT-1",
		CertPEM:      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		CAID:         "store-stage9-ca",
		Issuer:       "store-stage9",
		NotBefore:    now.Add(-1 * time.Hour),
		NotAfter:     now.Add(6 * time.Hour),
	})
	if err != nil {
		t.Fatalf("issue node cert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vmess', $3, 50, 70)
	`, tenant.ID, user.ID, now.Truncate(time.Hour)); err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	statusCounts, err := s.ListNodeStatusCounts(t.Context(), tenant.ID)
	if err != nil {
		t.Fatalf("list node status counts: %v", err)
	}
	if len(statusCounts) == 0 {
		t.Fatal("expected non-empty node status counts")
	}

	activeCerts, err := s.CountActiveNodeCertificates(t.Context(), tenant.ID)
	if err != nil {
		t.Fatalf("count active certs: %v", err)
	}
	if activeCerts < 1 {
		t.Fatalf("expected active certs >= 1, got %d", activeCerts)
	}

	expiringCerts, err := s.CountExpiringNodeCertificates(t.Context(), tenant.ID, now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("count expiring certs: %v", err)
	}
	if expiringCerts < 1 {
		t.Fatalf("expected expiring certs >= 1, got %d", expiringCerts)
	}

	traffic, err := s.GetTrafficUsageTotalBetween(t.Context(), tenant.ID, now.Add(-24*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("get traffic usage between: %v", err)
	}
	if traffic < 120 {
		t.Fatalf("expected traffic >= 120, got %d", traffic)
	}

	replayCount, err := s.CountAuditActionsSince(t.Context(), tenant.ID, "node.replay_rejected", now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("count replay audit actions: %v", err)
	}
	if replayCount < 1 {
		t.Fatalf("expected replay count >= 1, got %d", replayCount)
	}
}
