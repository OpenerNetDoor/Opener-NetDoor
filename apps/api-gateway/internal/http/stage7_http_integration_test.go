//go:build integration

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestGatewayNodeCertificatesLifecycleAndTenantIsolationWithLiveCore(t *testing.T) {
	databaseURL, migrationsDir := requireIntegrationDBConfig(t)
	db := openDB(t, databaseURL)
	applyMigrations(t, db, migrationsDir)
	resetData(t, db)

	coreAddr := allocateAddr(t)
	coreBaseURL := "http://" + coreAddr
	coreCmd := startCorePlatform(t, coreAddr, databaseURL)
	t.Cleanup(func() { shutdownCoreProcess(coreCmd) })
	waitHTTPReady(t, coreBaseURL+"/internal/ready", 20*time.Second)

	cfg := config.Config{
		HTTPAddr:            ":8080",
		CorePlatformBaseURL: coreBaseURL,
		JWTIssuer:           "iss",
		JWTAudience:         "aud",
		JWTSecret:           "very-secure-secret",
	}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new gateway handler: %v", err)
	}
	gw := httptest.NewServer(h)
	defer gw.Close()

	platformToken := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read", "admin:write", "platform:admin"},
	})
	platformHeaders := map[string]string{"Authorization": "Bearer " + platformToken}

	tenantA := gatewayCreateTenant(t, gw.URL, platformHeaders, uniqueName("tenant-gw-stage7-a"))
	tenantB := gatewayCreateTenant(t, gw.URL, platformHeaders, uniqueName("tenant-gw-stage7-b"))
	register := map[string]any{
		"tenant_id":        tenantB.ID,
		"region":           "eu-central",
		"hostname":         "node-gw-stage7-certs",
		"node_key_id":      "node-key-gw-stage7-certs",
		"node_public_key":  "pubkey-gw-stage7-certs",
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.0.0",
		"capabilities":     []string{"heartbeat.v1", "provisioning.v1"},
		"nonce":            "nonce-gw-stage7-register-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	register["signature"] = signGatewayRegister(register)

	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", platformHeaders, register)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 register node, got %d body=%s", status, body)
	}
	var registerResp struct {
		Node struct {
			ID string `json:"id"`
		} `json:"node"`
	}
	mustUnmarshal(t, body, &registerResp)

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/certificates/rotate", platformHeaders, map[string]any{
		"tenant_id": tenantB.ID,
		"node_id":   registerResp.Node.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 rotate cert, got %d body=%s", status, body)
	}

	status, body = gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/nodes/certificates?tenant_id="+tenantB.ID+"&node_id="+registerResp.Node.ID, platformHeaders, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 list certs, got %d body=%s", status, body)
	}

	tenantScopedToken := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read", "admin:write"},
		TenantID: tenantA.ID,
	})
	tenantScopedHeaders := map[string]string{"Authorization": "Bearer " + tenantScopedToken}
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/certificates/revoke", tenantScopedHeaders, map[string]any{
		"tenant_id": tenantB.ID,
		"node_id":   registerResp.Node.ID,
	})
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-tenant cert revoke, got %d body=%s", status, body)
	}
}
