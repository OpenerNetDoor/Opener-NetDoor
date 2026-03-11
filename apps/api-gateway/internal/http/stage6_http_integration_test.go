//go:build integration

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestGatewayNodeRevokeReactivateFlowWithLiveCore(t *testing.T) {
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

	token := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read", "admin:write", "platform:admin"},
	})
	headers := map[string]string{"Authorization": "Bearer " + token}

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw-stage6-node"))
	register := map[string]any{
		"tenant_id":        tenant.ID,
		"region":           "eu-central",
		"hostname":         "node-gw-stage6-1",
		"node_key_id":      "node-key-gw-stage6-1",
		"node_public_key":  "pubkey-gw-stage6-1",
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.2.0",
		"capabilities":     []string{"heartbeat.v1", "provisioning.v1"},
		"nonce":            "nonce-gw-stage6-register-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	register["signature"] = signGatewayRegister(register)
	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", headers, register)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 for node register, got %d body=%s", status, body)
	}

	var registerResp struct {
		Node struct {
			ID        string `json:"id"`
			NodeKeyID string `json:"node_key_id"`
		} `json:"node"`
		Provisioning struct {
			NodeCertificateSerial string `json:"node_certificate_serial"`
		} `json:"provisioning"`
	}
	mustUnmarshal(t, body, &registerResp)

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/revoke", headers, map[string]any{
		"tenant_id": tenant.ID,
		"node_id":   registerResp.Node.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 for node revoke, got %d body=%s", status, body)
	}

	hbDenied := map[string]any{
		"tenant_id":        tenant.ID,
		"node_id":          registerResp.Node.ID,
		"node_key_id":      registerResp.Node.NodeKeyID,
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.2.1",
		"tls_identity":     map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial},
		"nonce":            "nonce-gw-stage6-heartbeat-denied-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	hbDenied["signature"] = signGatewayHeartbeat(hbDenied, "pubkey-gw-stage6-1")
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/heartbeat", headers, hbDenied)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for heartbeat on revoked node, got %d body=%s", status, body)
	}

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/reactivate", headers, map[string]any{
		"tenant_id": tenant.ID,
		"node_id":   registerResp.Node.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200 for node reactivate, got %d body=%s", status, body)
	}

	hbAllowed := map[string]any{
		"tenant_id":        tenant.ID,
		"node_id":          registerResp.Node.ID,
		"node_key_id":      registerResp.Node.NodeKeyID,
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.2.2",
		"tls_identity":     map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial},
		"nonce":            "nonce-gw-stage6-heartbeat-ok-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	hbAllowed["signature"] = signGatewayHeartbeat(hbAllowed, "pubkey-gw-stage6-1")
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/heartbeat", headers, hbAllowed)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for heartbeat after reactivate, got %d body=%s", status, body)
	}
}

func TestGatewayNodeReplayProtectionWithLiveCore(t *testing.T) {
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

	token := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read", "admin:write", "platform:admin"},
	})
	headers := map[string]string{"Authorization": "Bearer " + token}

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw-stage6-replay"))
	register := map[string]any{
		"tenant_id":        tenant.ID,
		"region":           "eu-central",
		"hostname":         "node-gw-stage6-replay-1",
		"node_key_id":      "node-key-gw-stage6-replay-1",
		"node_public_key":  "pubkey-gw-stage6-replay-1",
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.2.0",
		"capabilities":     []string{"heartbeat.v1", "provisioning.v1"},
		"nonce":            "nonce-gw-stage6-replay-register-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	register["signature"] = signGatewayRegister(register)

	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", headers, register)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 on first register, got %d body=%s", status, body)
	}

	var registerResp struct {
		Node struct {
			ID        string `json:"id"`
			NodeKeyID string `json:"node_key_id"`
		} `json:"node"`
		Provisioning struct {
			NodeCertificateSerial string `json:"node_certificate_serial"`
		} `json:"provisioning"`
	}
	mustUnmarshal(t, body, &registerResp)

	registerReplay := cloneMap(register)
	registerReplay["tls_identity"] = map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial}
	registerReplay["signature"] = signGatewayRegister(registerReplay)
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", headers, registerReplay)
	if status != http.StatusConflict {
		t.Fatalf("expected 409 replay on second register, got %d body=%s", status, body)
	}

	register2 := cloneMap(register)
	register2["tls_identity"] = map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial}
	register2["nonce"] = "nonce-gw-stage6-replay-register-2"
	register2["signed_at"] = time.Now().UTC().Unix()
	register2["signature"] = signGatewayRegister(register2)
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", headers, register2)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 on third register with new nonce, got %d body=%s", status, body)
	}
	mustUnmarshal(t, body, &registerResp)

	hb := map[string]any{
		"tenant_id":        tenant.ID,
		"node_id":          registerResp.Node.ID,
		"node_key_id":      registerResp.Node.NodeKeyID,
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.2.1",
		"tls_identity":     map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial},
		"nonce":            "nonce-gw-stage6-replay-heartbeat-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	hb["signature"] = signGatewayHeartbeat(hb, "pubkey-gw-stage6-replay-1")

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/heartbeat", headers, hb)
	if status != http.StatusOK {
		t.Fatalf("expected 200 first heartbeat, got %d body=%s", status, body)
	}
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/heartbeat", headers, hb)
	if status != http.StatusConflict {
		t.Fatalf("expected 409 replay on second heartbeat, got %d body=%s", status, body)
	}
}

func mustUnmarshal(t *testing.T, payload string, out any) {
	t.Helper()
	if err := json.Unmarshal([]byte(payload), out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func cloneMap(src map[string]any) map[string]any {
	cp := make(map[string]any, len(src))
	for k, v := range src {
		cp[k] = v
	}
	return cp
}
