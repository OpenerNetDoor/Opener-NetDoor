//go:build integration

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

func TestHTTPNodeRevokeReactivateLifecycleWithPostgres(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	h := NewHandler(s)
	ts := httptest.NewServer(h)
	defer ts.Close()

	actorHeaders := map[string]string{
		"X-Actor-Sub":    "admin-platform-1",
		"X-Actor-Scopes": "admin:read,admin:write,platform:admin",
	}

	tenant := createTenant(t, ts.URL, actorHeaders, testutil.UniqueName("tenant-http-stage6"))
	registerReq := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-http-stage6-1",
		NodeKeyID:       "node-key-http-stage6-1",
		NodePublicKey:   "pubkey-http-stage6-1",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-http-stage6-register-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	registerReq.Signature = signRegisterNode(registerReq)
	registration := registerNode(t, ts.URL, actorHeaders, registerReq)

	revoked := nodeLifecycleRequest(t, ts.URL, actorHeaders, "/internal/v1/nodes/revoke", model.NodeLifecycleRequest{TenantID: tenant.ID, NodeID: registration.Node.ID})
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", revoked.Status)
	}

	hbDenied := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registration.Node.ID,
		NodeKeyID:       registration.Node.NodeKeyID,
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.1",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: registration.Provisioning.NodeCertificateSerial},
		Nonce:           "nonce-http-stage6-heartbeat-denied-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbDenied.Signature = signHeartbeatNode(hbDenied, registerReq.NodePublicKey)
	status, body := heartbeatNodeExpect(t, ts.URL, actorHeaders, hbDenied)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for revoked heartbeat, got %d body=%s", status, body)
	}

	reactivated := nodeLifecycleRequest(t, ts.URL, actorHeaders, "/internal/v1/nodes/reactivate", model.NodeLifecycleRequest{TenantID: tenant.ID, NodeID: registration.Node.ID})
	if reactivated.Status != "pending" {
		t.Fatalf("expected pending after reactivate, got %s", reactivated.Status)
	}

	hbAllowed := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registration.Node.ID,
		NodeKeyID:       registration.Node.NodeKeyID,
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.2",
		TLSIdentity:     &model.NodeTLSIdentity{SerialNumber: registration.Provisioning.NodeCertificateSerial},
		Nonce:           "nonce-http-stage6-heartbeat-ok-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	hbAllowed.Signature = signHeartbeatNode(hbAllowed, registerReq.NodePublicKey)
	node := heartbeatNode(t, ts.URL, actorHeaders, hbAllowed)
	if node.Status != "active" {
		t.Fatalf("expected active after reactivation heartbeat, got %s", node.Status)
	}
}

func TestHTTPNodeRegisterNonceRequired(t *testing.T) {
	databaseURL, migrationsDir := testutil.RequireDBConfig(t)
	db := testutil.OpenDB(t, databaseURL)
	testutil.ApplyMigrations(t, db, migrationsDir)
	testutil.ResetData(t, db)

	s, err := store.NewSQLStore(databaseURL)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer s.Close()

	h := NewHandler(s)
	ts := httptest.NewServer(h)
	defer ts.Close()

	actorHeaders := map[string]string{
		"X-Actor-Sub":    "admin-platform-1",
		"X-Actor-Scopes": "admin:read,admin:write,platform:admin",
	}

	tenant := createTenant(t, ts.URL, actorHeaders, testutil.UniqueName("tenant-http-stage6-nonce"))

	req := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-http-stage6-nonce",
		NodeKeyID:       "node-key-http-stage6-nonce",
		NodePublicKey:   "pubkey-http-stage6-nonce",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		SignedAt:        time.Now().UTC().Unix(),
	}
	req.Signature = signRegisterNode(req)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/nodes/register", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range actorHeaders {
		httpReq.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 400 for missing nonce, got %d body=%s", resp.StatusCode, buf.String())
	}
}

func nodeLifecycleRequest(t *testing.T, baseURL string, headers map[string]string, path string, in model.NodeLifecycleRequest) model.Node {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("node lifecycle request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 node lifecycle, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.Node
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode node lifecycle response: %v", err)
	}
	return out
}

func heartbeatNodeExpect(t *testing.T, baseURL string, headers map[string]string, in model.NodeHeartbeatRequest) (int, string) {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/nodes/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("node heartbeat request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String()
}
