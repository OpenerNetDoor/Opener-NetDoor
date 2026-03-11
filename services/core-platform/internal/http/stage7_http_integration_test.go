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

func TestHTTPNodeCertificatesLifecycleWithPostgres(t *testing.T) {
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

	headers := map[string]string{
		"X-Actor-Sub":    "admin-platform-1",
		"X-Actor-Scopes": "admin:read,admin:write,platform:admin",
	}

	tenant := createTenant(t, ts.URL, headers, testutil.UniqueName("tenant-http-stage7-certs"))
	registerReq := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-http-stage7-certs",
		NodeKeyID:       "node-key-http-stage7-certs",
		NodePublicKey:   "pubkey-http-stage7-certs",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-http-stage7-register-1",
		SignedAt:        time.Now().UTC().Unix(),
	}
	registerReq.Signature = signRegisterNode(registerReq)
	registration := registerNode(t, ts.URL, headers, registerReq)

	rotated := rotateNodeCertificateHTTP(t, ts.URL, headers, model.RotateNodeCertificateRequest{TenantID: tenant.ID, NodeID: registration.Node.ID})
	if rotated.ID == "" {
		t.Fatal("expected rotated certificate id")
	}

	certs := listNodeCertificatesHTTP(t, ts.URL, headers, tenant.ID, registration.Node.ID)
	if len(certs) == 0 {
		t.Fatal("expected non-empty certificate list")
	}

	revoked := revokeNodeCertificateHTTP(t, ts.URL, headers, model.RevokeNodeCertificateRequest{TenantID: tenant.ID, NodeID: registration.Node.ID, CertificateID: rotated.ID})
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at")
	}
}

func TestHTTPNodeCertificatesTenantIsolationDenyPath(t *testing.T) {
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

	platformHeaders := map[string]string{
		"X-Actor-Sub":    "admin-platform-1",
		"X-Actor-Scopes": "admin:read,admin:write,platform:admin",
	}
	tenantA := createTenant(t, ts.URL, platformHeaders, testutil.UniqueName("tenant-http-stage7-a"))
	tenantB := createTenant(t, ts.URL, platformHeaders, testutil.UniqueName("tenant-http-stage7-b"))

	registerReq := model.RegisterNodeRequest{
		TenantID:        tenantB.ID,
		Region:          "eu-central",
		Hostname:        "node-http-stage7-tenant-deny",
		NodeKeyID:       "node-key-http-stage7-tenant-deny",
		NodePublicKey:   "pubkey-http-stage7-tenant-deny",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-http-stage7-tenant-deny-register",
		SignedAt:        time.Now().UTC().Unix(),
	}
	registerReq.Signature = signRegisterNode(registerReq)
	registration := registerNode(t, ts.URL, platformHeaders, registerReq)

	tenantActorHeaders := map[string]string{
		"X-Actor-Sub":       "tenant-admin-a",
		"X-Actor-Scopes":    "admin:write",
		"X-Actor-Tenant-ID": tenantA.ID,
	}
	body, _ := json.Marshal(model.RotateNodeCertificateRequest{TenantID: tenantB.ID, NodeID: registration.Node.ID})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/nodes/certificates/rotate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range tenantActorHeaders {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rotate request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 403, got %d body=%s", resp.StatusCode, buf.String())
	}
}

func rotateNodeCertificateHTTP(t *testing.T, baseURL string, headers map[string]string, in model.RotateNodeCertificateRequest) model.NodeCertificate {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/nodes/certificates/rotate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rotate cert request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.NodeCertificate
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode rotate cert response: %v", err)
	}
	return out
}

func revokeNodeCertificateHTTP(t *testing.T, baseURL string, headers map[string]string, in model.RevokeNodeCertificateRequest) model.NodeCertificate {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/nodes/certificates/revoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revoke cert request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.NodeCertificate
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode revoke cert response: %v", err)
	}
	return out
}

func listNodeCertificatesHTTP(t *testing.T, baseURL string, headers map[string]string, tenantID string, nodeID string) []model.NodeCertificate {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/internal/v1/nodes/certificates?tenant_id="+tenantID+"&node_id="+nodeID, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list cert request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out struct {
		Items []model.NodeCertificate `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode cert list: %v", err)
	}
	return out.Items
}
