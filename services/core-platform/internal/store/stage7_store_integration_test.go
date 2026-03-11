//go:build integration

package store

import (
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestSQLStore_NodeCertificateLifecycleAndRotationLineage(t *testing.T) {
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
	now := time.Now().UTC()

	cert1, err := s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       node.ID,
		SerialNumber: "SERIAL-001",
		CertPEM:      "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		CAID:         "ca-stage7",
		Issuer:       "opener-netdoor-stage7-ca",
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("issue cert1: %v", err)
	}

	active, err := s.GetActiveNodeCertificate(t.Context(), tenant.ID, node.ID)
	if err != nil {
		t.Fatalf("get active cert: %v", err)
	}
	if active.ID != cert1.ID {
		t.Fatalf("expected active cert %s, got %s", cert1.ID, active.ID)
	}

	revoked, err := s.RevokeNodeCertificateByID(t.Context(), tenant.ID, node.ID, cert1.ID)
	if err != nil {
		t.Fatalf("revoke cert1: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}

	cert2, err := s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:         tenant.ID,
		NodeID:           node.ID,
		SerialNumber:     "SERIAL-002",
		CertPEM:          "-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----",
		CAID:             "ca-stage7",
		Issuer:           "opener-netdoor-stage7-ca",
		NotBefore:        now.Add(-1 * time.Minute),
		NotAfter:         now.Add(48 * time.Hour),
		RotateFromCertID: &cert1.ID,
	})
	if err != nil {
		t.Fatalf("issue cert2: %v", err)
	}
	if cert2.RotateFromCertID == nil || *cert2.RotateFromCertID != cert1.ID {
		t.Fatalf("expected rotate_from_cert_id=%s, got %+v", cert1.ID, cert2.RotateFromCertID)
	}

	active2, err := s.GetActiveNodeCertificate(t.Context(), tenant.ID, node.ID)
	if err != nil {
		t.Fatalf("get active cert after rotate: %v", err)
	}
	if active2.ID != cert2.ID {
		t.Fatalf("expected active cert %s, got %s", cert2.ID, active2.ID)
	}

	bySerial, err := s.GetNodeCertificateBySerial(t.Context(), tenant.ID, node.ID, cert2.SerialNumber)
	if err != nil {
		t.Fatalf("get by serial: %v", err)
	}
	if bySerial.ID != cert2.ID {
		t.Fatalf("expected cert2 by serial, got %s", bySerial.ID)
	}

	items, err := s.ListNodeCertificates(t.Context(), model.ListNodeCertificatesQuery{
		TenantID: tenant.ID,
		NodeID:   node.ID,
		ListQuery: model.ListQuery{
			Limit:  20,
			Offset: 0,
		},
	})
	if err != nil {
		t.Fatalf("list node certs: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 cert records, got %d", len(items))
	}
}

func createNodeForCertificateTests(t *testing.T, s *SQLStore) (model.Tenant, model.Node) {
	t.Helper()
	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant-cert-store")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	node, err := s.UpsertNodeRegistration(t.Context(), model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "store-cert-node-1",
		NodeKeyID:       "store-cert-node-key-1",
		NodePublicKey:   "store-cert-node-pub-1",
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "store-cert-register-nonce",
		SignedAt:        time.Now().UTC().Unix(),
	}, "fingerprint-store-cert-node")
	if err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	return tenant, node
}
