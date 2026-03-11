//go:build integration

package store

import (
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestSQLStore_PKIIssuerActivationLifecycle(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	issuer1, err := s.CreatePKIIssuer(t.Context(), model.CreatePKIIssuerRequest{
		IssuerID:   "issuer-stage8-a",
		Source:     "file",
		CAID:       "ca-stage8-a",
		IssuerName: "Stage8 A",
		CACertPEM:  "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
	})
	if err != nil {
		t.Fatalf("create issuer1: %v", err)
	}

	rotation1, err := s.ActivatePKIIssuer(t.Context(), issuer1.IssuerID)
	if err != nil {
		t.Fatalf("activate issuer1: %v", err)
	}
	if rotation1.ActiveIssuer.IssuerID != issuer1.IssuerID {
		t.Fatalf("expected issuer1 active, got %s", rotation1.ActiveIssuer.IssuerID)
	}
	if rotation1.PreviousIssuer != nil {
		t.Fatalf("expected no previous issuer, got %+v", rotation1.PreviousIssuer)
	}

	issuer2, err := s.CreatePKIIssuer(t.Context(), model.CreatePKIIssuerRequest{
		IssuerID:   "issuer-stage8-b",
		Source:     "file",
		CAID:       "ca-stage8-b",
		IssuerName: "Stage8 B",
		CACertPEM:  "-----BEGIN CERTIFICATE-----\nMIIC\n-----END CERTIFICATE-----",
	})
	if err != nil {
		t.Fatalf("create issuer2: %v", err)
	}

	rotation2, err := s.ActivatePKIIssuer(t.Context(), issuer2.IssuerID)
	if err != nil {
		t.Fatalf("activate issuer2: %v", err)
	}
	if rotation2.ActiveIssuer.IssuerID != issuer2.IssuerID {
		t.Fatalf("expected issuer2 active, got %s", rotation2.ActiveIssuer.IssuerID)
	}
	if rotation2.PreviousIssuer == nil || rotation2.PreviousIssuer.IssuerID != issuer1.IssuerID {
		t.Fatalf("expected issuer1 as previous, got %+v", rotation2.PreviousIssuer)
	}

	active, err := s.GetActivePKIIssuer(t.Context())
	if err != nil {
		t.Fatalf("get active issuer: %v", err)
	}
	if active.IssuerID != issuer2.IssuerID {
		t.Fatalf("expected issuer2 active, got %s", active.IssuerID)
	}
}

func TestSQLStore_ListNodeCertificatesExpiringBefore(t *testing.T) {
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
	node2, err := s.UpsertNodeRegistration(t.Context(), model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "store-cert-node-2",
		NodeKeyID:       "store-cert-node-key-2",
		NodePublicKey:   "store-cert-node-pub-2",
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "store-cert-register-nonce-2",
		SignedAt:        time.Now().UTC().Unix(),
	}, "fingerprint-store-cert-node-2")
	if err != nil {
		t.Fatalf("create second node: %v", err)
	}
	_ = node2
	now := time.Now().UTC()

	_, err = s.CreatePKIIssuer(t.Context(), model.CreatePKIIssuerRequest{
		IssuerID:   "issuer-stage8-expiring",
		Source:     "file",
		CAID:       "ca-stage8-expiring",
		IssuerName: "Stage8 Expiring",
		CACertPEM:  "-----BEGIN CERTIFICATE-----\nMIID\n-----END CERTIFICATE-----",
	})
	if err != nil {
		t.Fatalf("create issuer: %v", err)
	}

	_, err = s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       node.ID,
		SerialNumber: "SERIAL-EXP-1",
		CertPEM:      "-----BEGIN CERTIFICATE-----\nMIIE\n-----END CERTIFICATE-----",
		CAID:         "ca-stage8-expiring",
		IssuerID:     "issuer-stage8-expiring",
		Issuer:       "Stage8 Expiring",
		NotBefore:    now.Add(-1 * time.Hour),
		NotAfter:     now.Add(20 * time.Minute),
	})
	if err != nil {
		t.Fatalf("issue cert expiring soon: %v", err)
	}
	_, err = s.IssueNodeCertificate(t.Context(), model.IssueNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       node2.ID,
		SerialNumber: "SERIAL-EXP-2",
		CertPEM:      "-----BEGIN CERTIFICATE-----\nMIIF\n-----END CERTIFICATE-----",
		CAID:         "ca-stage8-expiring",
		IssuerID:     "issuer-stage8-expiring",
		Issuer:       "Stage8 Expiring",
		NotBefore:    now.Add(-1 * time.Hour),
		NotAfter:     now.Add(48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("issue cert long ttl: %v", err)
	}

	expiring, err := s.ListNodeCertificatesExpiringBefore(t.Context(), now.Add(1*time.Hour), 50)
	if err != nil {
		t.Fatalf("list expiring certs: %v", err)
	}
	if len(expiring) == 0 {
		t.Fatal("expected at least one expiring certificate")
	}
	if expiring[0].SerialNumber != "SERIAL-EXP-1" {
		t.Fatalf("expected SERIAL-EXP-1 first, got %s", expiring[0].SerialNumber)
	}
}
