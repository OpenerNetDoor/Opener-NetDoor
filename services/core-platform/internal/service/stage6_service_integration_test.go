//go:build integration

package service

import (
	"errors"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestCoreService_ReplayedRegistrationDenied(t *testing.T) {
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
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node-replay-register")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	req := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-replay-reg-1",
		NodeKeyID:       "node-key-replay-reg-1",
		NodePublicKey:   "pubkey-replay-reg-1",
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-register-replay-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	req.Signature = signRegister(req)

	registered, err := svc.RegisterNode(t.Context(), platform, req)
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	replayReq := req
	replayReq.TLSIdentity = &model.NodeTLSIdentity{SerialNumber: registered.Provisioning.NodeCertificateSerial}
	replayReq.Signature = signRegister(replayReq)
	if _, err := svc.RegisterNode(t.Context(), platform, replayReq); err == nil {
		t.Fatal("expected replay_detected on second register")
	} else {
		var appErr *AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T (%v)", err, err)
		}
		if appErr.Status != 409 || appErr.Code != "replay_detected" {
			t.Fatalf("expected 409/replay_detected, got %d/%s", appErr.Status, appErr.Code)
		}
	}
}

func TestCoreService_ReplayedHeartbeatDenied(t *testing.T) {
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
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node-replay-heartbeat")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	register := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-replay-hb-1",
		NodeKeyID:       "node-key-replay-hb-1",
		NodePublicKey:   "pubkey-replay-hb-1",
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-register-replay-hb-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	register.Signature = signRegister(register)
	registered, err := svc.RegisterNode(t.Context(), platform, register)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	hb := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       register.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: registered.Provisioning.NodeCertificateSerial},
		Nonce:           "nonce-heartbeat-replay-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hb.Signature = signHeartbeat(hb, register.NodePublicKey)

	if _, err := svc.NodeHeartbeat(t.Context(), platform, hb); err != nil {
		t.Fatalf("first heartbeat failed: %v", err)
	}
	if _, err := svc.NodeHeartbeat(t.Context(), platform, hb); err == nil {
		t.Fatal("expected replay_detected on second heartbeat")
	} else {
		var appErr *AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T (%v)", err, err)
		}
		if appErr.Status != 409 || appErr.Code != "replay_detected" {
			t.Fatalf("expected 409/replay_detected, got %d/%s", appErr.Status, appErr.Code)
		}
	}
}

func TestCoreService_RevokeReactivateLifecycle(t *testing.T) {
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
	tenant, err := svc.CreateTenant(t.Context(), platform, model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node-lifecycle")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	register := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-life-1",
		NodeKeyID:       "node-key-life-1",
		NodePublicKey:   "pubkey-life-1",
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-register-life-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	register.Signature = signRegister(register)
	registered, err := svc.RegisterNode(t.Context(), platform, register)
	if err != nil {
		t.Fatalf("register node: %v", err)
	}

	revoked, err := svc.RevokeNode(t.Context(), platform, model.NodeLifecycleRequest{TenantID: tenant.ID, NodeID: registered.Node.ID})
	if err != nil {
		t.Fatalf("revoke node: %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked, got %s", revoked.Status)
	}

	hbDenied := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       register.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.1",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: registered.Provisioning.NodeCertificateSerial},
		Nonce:           "nonce-heartbeat-denied-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbDenied.Signature = signHeartbeat(hbDenied, register.NodePublicKey)
	if _, err := svc.NodeHeartbeat(t.Context(), platform, hbDenied); err == nil {
		t.Fatal("expected node_revoked on heartbeat")
	} else {
		var appErr *AppError
		if !errors.As(err, &appErr) {
			t.Fatalf("expected AppError, got %T (%v)", err, err)
		}
		if appErr.Code != "node_revoked" {
			t.Fatalf("expected node_revoked, got %s", appErr.Code)
		}
	}

	reactivated, err := svc.ReactivateNode(t.Context(), platform, model.NodeLifecycleRequest{TenantID: tenant.ID, NodeID: registered.Node.ID})
	if err != nil {
		t.Fatalf("reactivate node: %v", err)
	}
	if reactivated.Status != "pending" {
		t.Fatalf("expected pending after reactivate, got %s", reactivated.Status)
	}

	hbAllowed := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registered.Node.ID,
		NodeKeyID:       register.NodeKeyID,
		ContractVersion: integrationNodeContractVersion,
		AgentVersion:    "1.0.2",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: registered.Provisioning.NodeCertificateSerial},
		Nonce:           "nonce-heartbeat-allowed-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbAllowed.Signature = signHeartbeat(hbAllowed, register.NodePublicKey)
	active, err := svc.NodeHeartbeat(t.Context(), platform, hbAllowed)
	if err != nil {
		t.Fatalf("heartbeat after reactivate: %v", err)
	}
	if active.Status != "active" {
		t.Fatalf("expected active after heartbeat, got %s", active.Status)
	}
}
