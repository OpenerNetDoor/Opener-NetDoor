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

func TestHTTPPKIIssuerEndpointsWithPostgres(t *testing.T) {
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

	body, _ := json.Marshal(model.CreatePKIIssuerRequest{IssuerID: "issuer-http-stage8", Activate: true})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/pki/issuers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range platformHeaders {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create pki issuer request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 201 create pki issuer, got %d body=%s", resp.StatusCode, buf.String())
	}

	activateBody, _ := json.Marshal(model.ActivatePKIIssuerRequest{IssuerID: "issuer-http-stage8"})
	activateReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/pki/issuers/activate", bytes.NewReader(activateBody))
	activateReq.Header.Set("Content-Type", "application/json")
	for k, v := range platformHeaders {
		activateReq.Header.Set(k, v)
	}
	activateResp, err := http.DefaultClient.Do(activateReq)
	if err != nil {
		t.Fatalf("activate pki issuer request: %v", err)
	}
	defer activateResp.Body.Close()
	if activateResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(activateResp.Body)
		t.Fatalf("expected 200 activate pki issuer, got %d body=%s", activateResp.StatusCode, buf.String())
	}

	listReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/pki/issuers", nil)
	for k, v := range platformHeaders {
		listReq.Header.Set(k, v)
	}
	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("list pki issuers request: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(listResp.Body)
		t.Fatalf("expected 200 list pki issuers, got %d body=%s", listResp.StatusCode, buf.String())
	}

	tenantHeaders := map[string]string{
		"X-Actor-Sub":       "tenant-admin",
		"X-Actor-Scopes":    "admin:read",
		"X-Actor-Tenant-ID": "tenant-a",
	}
	tenantReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/pki/issuers", nil)
	for k, v := range tenantHeaders {
		tenantReq.Header.Set(k, v)
	}
	tenantResp, err := http.DefaultClient.Do(tenantReq)
	if err != nil {
		t.Fatalf("tenant issuer list request: %v", err)
	}
	defer tenantResp.Body.Close()
	if tenantResp.StatusCode != http.StatusForbidden {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(tenantResp.Body)
		t.Fatalf("expected 403 list pki issuers for tenant actor, got %d body=%s", tenantResp.StatusCode, buf.String())
	}
}

func TestHTTPNodeCertificateRenewWithPostgres(t *testing.T) {
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

	tenant := createTenant(t, ts.URL, platformHeaders, testutil.UniqueName("tenant-http-stage8-renew"))
	registerReq := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-http-stage8-renew",
		NodeKeyID:       "node-key-http-stage8-renew",
		NodePublicKey:   "pubkey-http-stage8-renew",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.0.0",
		Capabilities:    []string{"heartbeat.v1", "provisioning.v1"},
		Nonce:           "nonce-http-stage8-renew-register",
		SignedAt:        time.Now().UTC().Unix(),
	}
	registerReq.Signature = signRegisterNode(registerReq)
	registration := registerNode(t, ts.URL, platformHeaders, registerReq)

	renewBody, _ := json.Marshal(model.RenewNodeCertificateRequest{TenantID: tenant.ID, NodeID: registration.Node.ID, Force: true})
	renewReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/nodes/certificates/renew", bytes.NewReader(renewBody))
	renewReq.Header.Set("Content-Type", "application/json")
	for k, v := range platformHeaders {
		renewReq.Header.Set(k, v)
	}
	renewResp, err := http.DefaultClient.Do(renewReq)
	if err != nil {
		t.Fatalf("renew cert request: %v", err)
	}
	defer renewResp.Body.Close()
	if renewResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(renewResp.Body)
		t.Fatalf("expected 200 renew cert, got %d body=%s", renewResp.StatusCode, buf.String())
	}
	var renewed model.RenewNodeCertificateResult
	if err := json.NewDecoder(renewResp.Body).Decode(&renewed); err != nil {
		t.Fatalf("decode renew response: %v", err)
	}
	if !renewed.Renewed {
		t.Fatal("expected renewed=true")
	}

	tenantHeaders := map[string]string{
		"X-Actor-Sub":       "tenant-admin-a",
		"X-Actor-Scopes":    "admin:write",
		"X-Actor-Tenant-ID": testutil.UniqueName("tenant-other"),
	}
	tenantReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/internal/v1/nodes/certificates/renew", bytes.NewReader(renewBody))
	tenantReq.Header.Set("Content-Type", "application/json")
	for k, v := range tenantHeaders {
		tenantReq.Header.Set(k, v)
	}
	tenantResp, err := http.DefaultClient.Do(tenantReq)
	if err != nil {
		t.Fatalf("tenant renew request: %v", err)
	}
	defer tenantResp.Body.Close()
	if tenantResp.StatusCode != http.StatusForbidden {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(tenantResp.Body)
		t.Fatalf("expected 403 renew cert for foreign tenant actor, got %d body=%s", tenantResp.StatusCode, buf.String())
	}
}
