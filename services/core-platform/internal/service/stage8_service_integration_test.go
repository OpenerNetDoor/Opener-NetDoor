//go:build integration

package service

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestCoreService_PKIIssuerLifecycleAndRenew(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s, Options{
		NodeSigningSecret:    integrationNodeSigningSecret,
		NodeContractVersion:  integrationNodeContractVersion,
		NodePKIMode:          "strict",
		NodeCAMode:           "file",
		NodeCAActiveIssuerID: "issuer-stage8-main",
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage8-pki")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	issuer, err := svc.CreatePKIIssuer(t.Context(), platform, model.CreatePKIIssuerRequest{IssuerID: "issuer-stage8-main", Activate: true})
	if err != nil {
		t.Fatalf("create issuer: %v", err)
	}
	if issuer.IssuerID != "issuer-stage8-main" {
		t.Fatalf("unexpected issuer id: %s", issuer.IssuerID)
	}

	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage8-renew")
	result, err := svc.RenewNodeCertificate(t.Context(), platform, model.RenewNodeCertificateRequest{TenantID: tenant.ID, NodeID: registered.Node.ID, Force: true})
	if err != nil {
		t.Fatalf("renew certificate: %v", err)
	}
	if !result.Renewed {
		t.Fatal("expected renewed=true")
	}
	if result.PreviousCertificateID == "" {
		t.Fatal("expected previous certificate id")
	}
	if result.Certificate.ID == "" {
		t.Fatal("expected new certificate id")
	}

	if count := auditActionGlobalCount(t, db, "issuer_create"); count < 1 {
		t.Fatalf("expected issuer_create audit row, got %d", count)
	}
	if count := auditActionGlobalCount(t, db, "issuer_activate"); count < 1 {
		t.Fatalf("expected issuer_activate audit row, got %d", count)
	}
	if count := auditActionCount(t, db, tenant.ID, "cert_renew"); count < 1 {
		t.Fatalf("expected cert_renew audit row, got %d", count)
	}
}

func TestCoreService_TrustedPreviousIssuerAcceptedThenRejected(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s, Options{
		NodeSigningSecret:       integrationNodeSigningSecret,
		NodeContractVersion:     integrationNodeContractVersion,
		NodePKIMode:             "strict",
		NodeCAMode:              "file",
		NodeCAActiveIssuerID:    "issuer-stage8-a",
		NodeCAPreviousIssuerIDs: []string{"issuer-stage8-a"},
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage8-trust")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	_, err = svc.CreatePKIIssuer(t.Context(), platform, model.CreatePKIIssuerRequest{IssuerID: "issuer-stage8-a", Activate: true})
	if err != nil {
		t.Fatalf("activate issuer a: %v", err)
	}
	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage8-trust")
	oldSerial := registered.Provisioning.NodeCertificateSerial

	_, err = svc.CreatePKIIssuer(t.Context(), platform, model.CreatePKIIssuerRequest{IssuerID: "issuer-stage8-b", Source: "external", CAID: "issuer-stage8-b-ca", CACertPEM: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----", Activate: true})
	if err != nil {
		t.Fatalf("activate issuer b: %v", err)
	}

	core := svc
	core.opts.NodeCAActiveIssuerID = "issuer-stage8-b"
	core.opts.NodeCAPreviousIssuerIDs = []string{"issuer-stage8-a"}

	hbAllowed := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: oldSerial},
		Nonce:           "nonce-stage8-trusted-allowed",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbAllowed.Signature = signHeartbeat(hbAllowed, registered.Node.NodePublicKey)
	if _, err := svc.NodeHeartbeat(t.Context(), platform, hbAllowed); err != nil {
		t.Fatalf("heartbeat with trusted previous issuer should pass: %v", err)
	}

	core.opts.NodeCAPreviousIssuerIDs = nil

	hbDenied := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.2",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: oldSerial},
		Nonce:           "nonce-stage8-trusted-denied",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbDenied.Signature = signHeartbeat(hbDenied, registered.Node.NodePublicKey)
	_, err = svc.NodeHeartbeat(t.Context(), platform, hbDenied)
	if err == nil {
		t.Fatal("expected untrusted issuer error after trust window closed")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != "node_certificate_untrusted_issuer" {
		t.Fatalf("expected node_certificate_untrusted_issuer, got %s", appErr.Code)
	}
}

func TestCoreService_TenantScopedActorCannotActivateIssuer(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	svc := New(s, Options{
		NodeSigningSecret:   integrationNodeSigningSecret,
		NodeContractVersion: integrationNodeContractVersion,
		NodePKIMode:         "strict",
		NodeCAMode:          "file",
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage8-deny")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	tenantActor := testutil.TenantActor(tenant.ID)

	_, err = svc.ActivatePKIIssuer(t.Context(), tenantActor, model.ActivatePKIIssuerRequest{IssuerID: "issuer-stage8-x"})
	if err == nil {
		t.Fatal("expected forbidden for tenant scoped actor")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != "forbidden" {
		t.Fatalf("expected forbidden, got %s", appErr.Code)
	}
}

func auditActionGlobalCount(t *testing.T, db *sql.DB, action string) int {
	t.Helper()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = $1`, action).Scan(&count)
	if err != nil {
		t.Fatalf("query global audit count: %v", err)
	}
	return count
}
