//go:build integration

package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestSQLStore_ConsumeNodeNonceReplayPersistence(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node-replay")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	nonceReq := model.ConsumeNodeNonceRequest{
		TenantID:    tenant.ID,
		NodeKeyID:   "node-key-1",
		RequestType: "register",
		Nonce:       "nonce-stage6-12345",
		SignedAt:    time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(5 * time.Minute),
	}

	if err := s.ConsumeNodeNonce(t.Context(), nonceReq); err != nil {
		t.Fatalf("consume nonce first: %v", err)
	}
	if err := s.ConsumeNodeNonce(t.Context(), nonceReq); err == nil {
		t.Fatal("expected replay conflict on duplicate nonce")
	} else {
		var dbErr *DBError
		if !errors.As(err, &dbErr) {
			t.Fatalf("expected DBError, got %T (%v)", err, err)
		}
		if dbErr.Kind != ErrorKindConflict {
			t.Fatalf("expected conflict kind, got %s", dbErr.Kind)
		}
	}
}

func TestSQLStore_NodeRevokeAndReactivateLifecycle(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	tenant, err := s.CreateTenant(t.Context(), model.CreateTenantRequest{Name: testutil.UniqueName("tenant-node-lifecycle")})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	node, err := s.UpsertNodeRegistration(t.Context(), model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu",
		Hostname:        "node-1",
		NodeKeyID:       "node-key-1",
		NodePublicKey:   "pub-1",
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-register-1",
		SignedAt:        time.Now().UTC().Unix(),
	}, "fp-1")
	if err != nil {
		t.Fatalf("upsert node: %v", err)
	}

	revoked, err := s.RevokeNode(t.Context(), tenant.ID, node.ID)
	if err != nil {
		t.Fatalf("revoke node: %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked, got %s", revoked.Status)
	}

	_, err = s.TouchNodeHeartbeat(t.Context(), model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          node.ID,
		NodeKeyID:       node.NodeKeyID,
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.1",
		Nonce:           "nonce-heartbeat-denied",
		SignedAt:        time.Now().UTC().Unix(),
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for revoked heartbeat, got %v", err)
	}

	reactivated, err := s.ReactivateNode(t.Context(), tenant.ID, node.ID)
	if err != nil {
		t.Fatalf("reactivate node: %v", err)
	}
	if reactivated.Status != "pending" {
		t.Fatalf("expected pending after reactivate, got %s", reactivated.Status)
	}
	if reactivated.LastHeartbeatAt != nil {
		t.Fatal("expected nil last_heartbeat_at after reactivate")
	}

	active, err := s.TouchNodeHeartbeat(t.Context(), model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          node.ID,
		NodeKeyID:       node.NodeKeyID,
		ContractVersion: "2026-03-10.stage5.v1",
		AgentVersion:    "1.0.2",
		Nonce:           "nonce-heartbeat-ok",
		SignedAt:        time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatalf("touch heartbeat after reactivate: %v", err)
	}
	if active.Status != "active" {
		t.Fatalf("expected active after heartbeat, got %s", active.Status)
	}
}
