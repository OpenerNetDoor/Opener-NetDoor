//go:build integration

package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestCoreService_StrictPKIDefaultRejectsHeartbeatWithoutTLSIdentity(t *testing.T) {
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
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-strict")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage7-strict")
	hb := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		Nonce:           "hb-stage7-strict-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hb.Signature = signHeartbeat(hb, registered.Node.NodePublicKey)

	_, err = svc.NodeHeartbeat(t.Context(), platform, hb)
	if err == nil {
		t.Fatal("expected invalid_node_certificate in strict mode")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != "invalid_node_certificate" {
		t.Fatalf("expected invalid_node_certificate, got %s", appErr.Code)
	}
}

func TestCoreService_LegacyFallbackAllowsHeartbeatWithoutTLSIdentityWhenEnabled(t *testing.T) {
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
		NodeSigningSecret:      integrationNodeSigningSecret,
		NodeContractVersion:    integrationNodeContractVersion,
		NodePKIMode:            "strict",
		NodeLegacyHMACFallback: true,
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-fallback")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage7-fallback")
	hb := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		Nonce:           "hb-stage7-fallback-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hb.Signature = signHeartbeat(hb, registered.Node.NodePublicKey)

	node, err := svc.NodeHeartbeat(t.Context(), platform, hb)
	if err != nil {
		t.Fatalf("heartbeat with fallback enabled: %v", err)
	}
	if node.Status != "active" {
		t.Fatalf("expected active, got %s", node.Status)
	}
}

func TestCoreService_CertificateRotationRevocationAndAudit(t *testing.T) {
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
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-rotate")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage7-rotate")
	activeBefore, err := activeCertificateForNode(t.Context(), svc, platform, tenant.ID, registered.Node.ID)
	if err != nil {
		t.Fatalf("active cert before rotate: %v", err)
	}

	rotated, err := svc.RotateNodeCertificate(t.Context(), platform, model.RotateNodeCertificateRequest{TenantID: tenant.ID, NodeID: registered.Node.ID})
	if err != nil {
		t.Fatalf("rotate cert: %v", err)
	}
	if rotated.RotateFromCertID == nil || *rotated.RotateFromCertID != activeBefore.ID {
		t.Fatalf("expected rotate_from=%s, got %+v", activeBefore.ID, rotated.RotateFromCertID)
	}

	revoked, err := svc.RevokeNodeCertificate(t.Context(), platform, model.RevokeNodeCertificateRequest{
		TenantID:      tenant.ID,
		NodeID:        registered.Node.ID,
		CertificateID: rotated.ID,
	})
	if err != nil {
		t.Fatalf("revoke cert: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}

	if count := auditActionCount(t, db, tenant.ID, "node.certificate_rotated"); count < 1 {
		t.Fatalf("expected audit row for node.certificate_rotated, got %d", count)
	}
	if count := auditActionCount(t, db, tenant.ID, "node.certificate_revoked"); count < 1 {
		t.Fatalf("expected audit row for node.certificate_revoked, got %d", count)
	}
}

func TestCoreService_ExpiredAndRevokedCertificatesAreRejected(t *testing.T) {
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
	})
	platform := testutil.PlatformAdminActor()
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-cert-check")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	registered := registerNodeForStage7(t, svc, platform, tenant.ID, "stage7-cert-check")
	active, err := activeCertificateForNode(t.Context(), svc, platform, tenant.ID, registered.Node.ID)
	if err != nil {
		t.Fatalf("active cert lookup: %v", err)
	}

	_, err = db.Exec(`UPDATE node_certificates SET not_after = NOW() - INTERVAL '1 minute' WHERE id = $1`, active.ID)
	if err != nil {
		t.Fatalf("expire cert manually: %v", err)
	}

	hbExpired := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.2",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: active.SerialNumber},
		Nonce:           "hb-stage7-expired-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbExpired.Signature = signHeartbeat(hbExpired, registered.Node.NodePublicKey)
	_, err = svc.NodeHeartbeat(t.Context(), platform, hbExpired)
	if err == nil {
		t.Fatal("expected node_certificate_expired")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "node_certificate_expired" {
		t.Fatalf("expected node_certificate_expired, got %v", err)
	}

	rotated, err := svc.RotateNodeCertificate(t.Context(), platform, model.RotateNodeCertificateRequest{TenantID: tenant.ID, NodeID: registered.Node.ID})
	if err != nil {
		t.Fatalf("rotate after expire: %v", err)
	}

	_, err = svc.RevokeNodeCertificate(t.Context(), platform, model.RevokeNodeCertificateRequest{
		TenantID:     tenant.ID,
		NodeID:       registered.Node.ID,
		SerialNumber: rotated.SerialNumber,
	})
	if err != nil {
		t.Fatalf("revoke rotated cert: %v", err)
	}

	hbRevoked := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       registered.Node.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.3",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: rotated.SerialNumber},
		Nonce:           "hb-stage7-revoked-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbRevoked.Signature = signHeartbeat(hbRevoked, registered.Node.NodePublicKey)
	_, err = svc.NodeHeartbeat(t.Context(), platform, hbRevoked)
	if err == nil {
		t.Fatal("expected node_certificate_revoked")
	}
	if !errors.As(err, &appErr) || appErr.Code != "node_certificate_revoked" {
		t.Fatalf("expected node_certificate_revoked, got %v", err)
	}

	if count := auditActionCount(t, db, tenant.ID, "node.certificate_rejected"); count < 2 {
		t.Fatalf("expected audit rows for rejected certificates, got %d", count)
	}
}

func TestCoreService_TenantScopedActorCannotRotateForeignTenantCertificate(t *testing.T) {
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
	})
	platform := testutil.PlatformAdminActor()
	tenantA, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-a")})
	if err != nil {
		t.Fatalf("create tenant a: %v", err)
	}
	tenantB, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-stage7-b")})
	if err != nil {
		t.Fatalf("create tenant b: %v", err)
	}

	registered := registerNodeForStage7(t, svc, platform, tenantB.ID, "stage7-tenant-deny")
	tenantScopedActor := testutil.TenantActor(tenantA.ID)

	_, err = svc.RotateNodeCertificate(t.Context(), tenantScopedActor, model.RotateNodeCertificateRequest{
		TenantID: tenantB.ID,
		NodeID:   registered.Node.ID,
	})
	if err == nil {
		t.Fatal("expected forbidden for cross-tenant rotate")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "forbidden" {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func registerNodeForStage7(t *testing.T, svc Service, actor model.ActorPrincipal, tenantID string, suffix string) model.NodeRegistrationResult {
	t.Helper()
	register := model.RegisterNodeRequest{
		TenantID:        tenantID,
		Region:          "eu-central",
		Hostname:        "node-" + suffix,
		NodeKeyID:       "node-key-" + suffix,
		NodePublicKey:   "pubkey-" + suffix,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "register-nonce-" + suffix,
		SignedAt:        time.Now().UTC().Unix(),
	}
	register.Signature = signRegister(register)
	registered, err := svc.RegisterNode(t.Context(), actor, register)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}
	if registered.Node.ID == "" {
		t.Fatal("expected node id")
	}
	if registered.Provisioning.NodeCertificateSerial == "" {
		t.Fatal("expected node certificate serial in provisioning")
	}
	return registered
}

func activeCertificateForNode(ctx context.Context, svc Service, actor model.ActorPrincipal, tenantID string, nodeID string) (model.NodeCertificate, error) {
	certs, err := svc.ListNodeCertificates(ctx, actor, model.ListNodeCertificatesQuery{
		TenantID: tenantID,
		NodeID:   nodeID,
		Status:   "active",
		ListQuery: model.ListQuery{
			Limit:  20,
			Offset: 0,
		},
	})
	if err != nil {
		return model.NodeCertificate{}, err
	}
	if len(certs) == 0 {
		return model.NodeCertificate{}, sql.ErrNoRows
	}
	return certs[0], nil
}

func auditActionCount(t *testing.T, db *sql.DB, tenantID string, action string) int {
	t.Helper()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action = $2`, tenantID, action).Scan(&count)
	if err != nil {
		t.Fatalf("query audit count: %v", err)
	}
	return count
}
