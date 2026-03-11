//go:build integration

package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

const (
	gatewayNodeSigningSecret   = "opener-netdoor-stage5-dev-signing-secret"
	gatewayNodeContractVersion = "2026-03-10.stage5.v1"
)

func TestGatewayAdminPolicyFlowWithLiveCore(t *testing.T) {
	databaseURL, migrationsDir := requireIntegrationDBConfig(t)
	db := openDB(t, databaseURL)
	applyMigrations(t, db, migrationsDir)
	resetData(t, db)

	coreAddr := allocateAddr(t)
	coreBaseURL := "http://" + coreAddr
	coreCmd := startCorePlatform(t, coreAddr, databaseURL)
	t.Cleanup(func() {
		shutdownCoreProcess(coreCmd)
	})
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

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw"))
	user := gatewayCreateUser(t, gw.URL, headers, tenant.ID, "gw-user@example.com")

	quota := int64(1000)
	deviceLimit := 1
	ttl := 300
	gatewaySetTenantPolicy(t, gw.URL, headers, map[string]any{
		"tenant_id":               tenant.ID,
		"traffic_quota_bytes":     quota,
		"device_limit":            deviceLimit,
		"default_key_ttl_seconds": ttl,
	})

	effective := gatewayGetEffectivePolicy(t, gw.URL, headers, tenant.ID, user.ID)
	if effective.Source == "" {
		t.Fatal("expected non-empty effective policy source")
	}

	gatewayRegisterDevice(t, gw.URL, headers, map[string]any{
		"tenant_id":          tenant.ID,
		"user_id":            user.ID,
		"device_fingerprint": "gw-fp-1",
		"platform":           "android",
	})

	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/devices/register", headers, map[string]any{
		"tenant_id":          tenant.ID,
		"user_id":            user.ID,
		"device_fingerprint": "gw-fp-2",
		"platform":           "ios",
	})
	if status != http.StatusConflict {
		t.Fatalf("expected 409 for second device, got %d body=%s", status, body)
	}

	_, err = db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 700, 700)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour))
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/access-keys", headers, map[string]any{
		"tenant_id": tenant.ID,
		"user_id":   user.ID,
		"key_type":  "vless",
	})
	if status != http.StatusConflict {
		t.Fatalf("expected 409 for quota exceeded, got %d body=%s", status, body)
	}
}

func TestGatewayUsersAndKeysOwnerFlowWithLiveCore(t *testing.T) {
	databaseURL, migrationsDir := requireIntegrationDBConfig(t)
	db := openDB(t, databaseURL)
	applyMigrations(t, db, migrationsDir)
	resetData(t, db)

	coreAddr := allocateAddr(t)
	coreBaseURL := "http://" + coreAddr
	coreCmd := startCorePlatform(t, coreAddr, databaseURL)
	t.Cleanup(func() {
		shutdownCoreProcess(coreCmd)
	})
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

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw-users"))
	user := gatewayCreateUser(t, gw.URL, headers, tenant.ID, "gw-owner-user@example.com")

	users := gatewayListUsers(t, gw.URL, headers, tenant.ID)
	if len(users) != 1 || users[0].ID != user.ID {
		t.Fatalf("expected created user in list, got %+v", users)
	}

	blocked := gatewaySetUserStatus(t, gw.URL, headers, "/v1/admin/users/block", map[string]any{"tenant_id": tenant.ID, "user_id": user.ID})
	if blocked.Status != "blocked" {
		t.Fatalf("expected blocked status, got %s", blocked.Status)
	}

	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/access-keys", headers, map[string]any{"tenant_id": tenant.ID, "user_id": user.ID, "key_type": "vless"})
	if status != http.StatusConflict {
		t.Fatalf("expected 409 for blocked user key create, got %d body=%s", status, body)
	}

	unblocked := gatewaySetUserStatus(t, gw.URL, headers, "/v1/admin/users/unblock", map[string]any{"tenant_id": tenant.ID, "user_id": user.ID})
	if unblocked.Status != "active" {
		t.Fatalf("expected active status after unblock, got %s", unblocked.Status)
	}

	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/access-keys", headers, map[string]any{"tenant_id": tenant.ID, "user_id": user.ID, "key_type": "vless"})
	if status != http.StatusCreated {
		t.Fatalf("expected 201 for key create, got %d body=%s", status, body)
	}
	var createdKey struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &createdKey); err != nil {
		t.Fatalf("decode created key: %v", err)
	}

	status, body = gatewayRequest(t, http.MethodDelete, gw.URL+"/v1/admin/access-keys?id="+createdKey.ID+"&tenant_id="+tenant.ID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for key revoke, got %d body=%s", status, body)
	}

	status, body = gatewayRequest(t, http.MethodDelete, gw.URL+"/v1/admin/users?id="+user.ID+"&tenant_id="+tenant.ID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for user delete, got %d body=%s", status, body)
	}

	users = gatewayListUsers(t, gw.URL, headers, tenant.ID)
	if len(users) != 0 {
		t.Fatalf("expected no users after delete, got %+v", users)
	}
}

func TestGatewayServerOwnerLifecycleWithLiveCore(t *testing.T) {
	databaseURL, migrationsDir := requireIntegrationDBConfig(t)
	db := openDB(t, databaseURL)
	applyMigrations(t, db, migrationsDir)
	resetData(t, db)

	coreAddr := allocateAddr(t)
	coreBaseURL := "http://" + coreAddr
	coreCmd := startCorePlatform(t, coreAddr, databaseURL)
	t.Cleanup(func() {
		shutdownCoreProcess(coreCmd)
	})
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

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw-server-owner"))
	created := gatewayCreateNode(t, gw.URL, headers, map[string]any{
		"tenant_id":    tenant.ID,
		"region":       "de-central",
		"hostname":     "de-owner-1.example.com",
		"capabilities": []string{"heartbeat.v1", "provisioning.v1"},
	})
	if created.Status != "pending" {
		t.Fatalf("expected pending status for created node, got %s", created.Status)
	}

	nodes := gatewayListNodes(t, gw.URL, headers, tenant.ID)
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}

	detail := gatewayGetNodeDetail(t, gw.URL, headers, tenant.ID, created.ID)
	if detail.ID != created.ID {
		t.Fatalf("expected node detail id %s, got %s", created.ID, detail.ID)
	}

	revoked := gatewayNodeLifecycle(t, gw.URL, headers, "/v1/admin/nodes/revoke", map[string]any{"tenant_id": tenant.ID, "node_id": created.ID})
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", revoked.Status)
	}

	reactivated := gatewayNodeLifecycle(t, gw.URL, headers, "/v1/admin/nodes/reactivate", map[string]any{"tenant_id": tenant.ID, "node_id": created.ID})
	if reactivated.Status != "pending" {
		t.Fatalf("expected pending status after reactivate, got %s", reactivated.Status)
	}
}
func TestGatewayNodeRegistrationHeartbeatFlowWithLiveCore(t *testing.T) {
	databaseURL, migrationsDir := requireIntegrationDBConfig(t)
	db := openDB(t, databaseURL)
	applyMigrations(t, db, migrationsDir)
	resetData(t, db)

	coreAddr := allocateAddr(t)
	coreBaseURL := "http://" + coreAddr
	coreCmd := startCorePlatform(t, coreAddr, databaseURL)
	t.Cleanup(func() {
		shutdownCoreProcess(coreCmd)
	})
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

	tenant := gatewayCreateTenant(t, gw.URL, headers, uniqueName("tenant-gw-node"))
	register := map[string]any{
		"tenant_id":        tenant.ID,
		"region":           "eu-central",
		"hostname":         "node-gw-1",
		"node_key_id":      "node-key-gw-1",
		"node_public_key":  "pubkey-gw-1",
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.1.0",
		"capabilities":     []string{"heartbeat.v1", "provisioning.v1"},
		"nonce":            "nonce-gw-register-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	register["signature"] = signGatewayRegister(register)

	status, body := gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/register", headers, register)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 for node register, got %d body=%s", status, body)
	}
	var registerResp struct {
		Node struct {
			ID         string `json:"id"`
			TenantID   string `json:"tenant_id"`
			NodeKeyID  string `json:"node_key_id"`
			Status     string `json:"status"`
			NodePubKey string `json:"node_public_key"`
		} `json:"node"`
		Provisioning struct {
			NodeCertificateSerial string `json:"node_certificate_serial"`
		} `json:"provisioning"`
	}
	if err := json.Unmarshal([]byte(body), &registerResp); err != nil {
		t.Fatalf("decode node register response: %v", err)
	}
	if strings.TrimSpace(registerResp.Node.ID) == "" {
		t.Fatal("expected node id in register response")
	}

	heartbeat := map[string]any{
		"tenant_id":        tenant.ID,
		"node_id":          registerResp.Node.ID,
		"node_key_id":      registerResp.Node.NodeKeyID,
		"contract_version": gatewayNodeContractVersion,
		"agent_version":    "1.1.1",
		"tls_identity":     map[string]any{"serial_number": registerResp.Provisioning.NodeCertificateSerial},
		"nonce":            "nonce-gw-heartbeat-1",
		"signed_at":        time.Now().UTC().Unix(),
	}
	heartbeat["signature"] = signGatewayHeartbeat(heartbeat, "pubkey-gw-1")
	status, body = gatewayRequest(t, http.MethodPost, gw.URL+"/v1/admin/nodes/heartbeat", headers, heartbeat)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for node heartbeat, got %d body=%s", status, body)
	}

	status, body = gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/nodes?tenant_id="+tenant.ID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for node list, got %d body=%s", status, body)
	}

	status, body = gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/nodes/provisioning?tenant_id="+tenant.ID+"&node_id="+registerResp.Node.ID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for node provisioning, got %d body=%s", status, body)
	}
}

type gatewayTenant struct {
	ID string `json:"id"`
}

type gatewayUser struct {
	ID string `json:"id"`
}

type gatewayUserRecord struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type gatewayEffectivePolicy struct {
	Source string `json:"source"`
}
type gatewayNodeRecord struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func gatewayCreateTenant(t *testing.T, baseURL string, headers map[string]string, name string) gatewayTenant {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+"/v1/admin/tenants", headers, map[string]any{"name": name})
	if status != http.StatusCreated {
		t.Fatalf("create tenant expected 201, got %d body=%s", status, body)
	}
	var out gatewayTenant
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode tenant: %v", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		t.Fatal("tenant id is empty")
	}
	return out
}

func gatewayCreateUser(t *testing.T, baseURL string, headers map[string]string, tenantID string, email string) gatewayUser {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+"/v1/admin/users", headers, map[string]any{"tenant_id": tenantID, "email": email})
	if status != http.StatusCreated {
		t.Fatalf("create user expected 201, got %d body=%s", status, body)
	}
	var out gatewayUser
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		t.Fatal("user id is empty")
	}
	return out
}

func gatewayCreateNode(t *testing.T, baseURL string, headers map[string]string, payload map[string]any) gatewayNodeRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+"/v1/admin/nodes", headers, payload)
	if status != http.StatusCreated {
		t.Fatalf("create node expected 201, got %d body=%s", status, body)
	}
	var out gatewayNodeRecord
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode node create: %v", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		t.Fatal("created node id is empty")
	}
	return out
}

func gatewayListNodes(t *testing.T, baseURL string, headers map[string]string, tenantID string) []gatewayNodeRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodGet, baseURL+"/v1/admin/nodes?tenant_id="+tenantID+"&limit=50&offset=0", headers, nil)
	if status != http.StatusOK {
		t.Fatalf("list nodes expected 200, got %d body=%s", status, body)
	}
	var out struct {
		Items []gatewayNodeRecord `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode nodes list: %v", err)
	}
	return out.Items
}

func gatewayGetNodeDetail(t *testing.T, baseURL string, headers map[string]string, tenantID string, nodeID string) gatewayNodeRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodGet, baseURL+"/v1/admin/nodes/detail?tenant_id="+tenantID+"&node_id="+nodeID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("node detail expected 200, got %d body=%s", status, body)
	}
	var out gatewayNodeRecord
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode node detail: %v", err)
	}
	return out
}

func gatewayNodeLifecycle(t *testing.T, baseURL string, headers map[string]string, path string, payload map[string]any) gatewayNodeRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+path, headers, payload)
	if status != http.StatusOK {
		t.Fatalf("node lifecycle expected 200, got %d body=%s", status, body)
	}
	var out gatewayNodeRecord
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode node lifecycle response: %v", err)
	}
	return out
}
func gatewayListUsers(t *testing.T, baseURL string, headers map[string]string, tenantID string) []gatewayUserRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodGet, baseURL+"/v1/admin/users?tenant_id="+tenantID+"&limit=50&offset=0", headers, nil)
	if status != http.StatusOK {
		t.Fatalf("list users expected 200, got %d body=%s", status, body)
	}
	var out struct {
		Items []gatewayUserRecord `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode users list: %v", err)
	}
	return out.Items
}

func gatewaySetUserStatus(t *testing.T, baseURL string, headers map[string]string, endpoint string, payload map[string]any) gatewayUserRecord {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+endpoint, headers, payload)
	if status != http.StatusOK {
		t.Fatalf("set user status expected 200, got %d body=%s", status, body)
	}
	var out gatewayUserRecord
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode user status response: %v", err)
	}
	return out
}

func gatewaySetTenantPolicy(t *testing.T, baseURL string, headers map[string]string, payload map[string]any) {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPut, baseURL+"/v1/admin/policies/tenants", headers, payload)
	if status != http.StatusOK {
		t.Fatalf("set tenant policy expected 200, got %d body=%s", status, body)
	}
}

func gatewayGetEffectivePolicy(t *testing.T, baseURL string, headers map[string]string, tenantID string, userID string) gatewayEffectivePolicy {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodGet, baseURL+"/v1/admin/policies/effective?tenant_id="+tenantID+"&user_id="+userID, headers, nil)
	if status != http.StatusOK {
		t.Fatalf("get effective policy expected 200, got %d body=%s", status, body)
	}
	var out gatewayEffectivePolicy
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode effective policy: %v", err)
	}
	return out
}

func gatewayRegisterDevice(t *testing.T, baseURL string, headers map[string]string, payload map[string]any) {
	t.Helper()
	status, body := gatewayRequest(t, http.MethodPost, baseURL+"/v1/admin/devices/register", headers, payload)
	if status != http.StatusCreated {
		t.Fatalf("register device expected 201, got %d body=%s", status, body)
	}
}

func gatewayRequest(t *testing.T, method string, url string, headers map[string]string, payload any) (int, string) {
	t.Helper()
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, bodyReader)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String()
}

func requireIntegrationDBConfig(t *testing.T) (databaseURL string, migrationsDir string) {
	t.Helper()
	databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	migrationsDir = strings.TrimSpace(os.Getenv("TEST_MIGRATIONS_DIR"))
	if databaseURL == "" || migrationsDir == "" {
		t.Skip("integration env is not set: TEST_DATABASE_URL and TEST_MIGRATIONS_DIR are required")
	}
	return databaseURL, migrationsDir
}

func openDB(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

func applyMigrations(t *testing.T, db *sql.DB, migrationsDir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no migrations in %s", migrationsDir)
	}
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("apply migration %s: %v", f, err)
		}
	}
}

func resetData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		TRUNCATE TABLE
		  node_heartbeats,
		  node_request_nonces,
		  node_certificates,
		  pki_issuers,
		  audit_logs,
		  traffic_usage_hourly,
		  user_policy_overrides,
		  tenant_policies,
		  access_keys,
		  devices,
		  nodes,
		  admins,
		  users,
		  tenants
		RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("reset data: %v", err)
	}
}

func allocateAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate addr: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func startCorePlatform(t *testing.T, httpAddr string, databaseURL string) *exec.Cmd {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	cmd := exec.Command("go", "run", "./services/core-platform/cmd/core-platform")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"HTTP_ADDR="+httpAddr,
		"DATABASE_URL="+databaseURL,
		"NODE_SIGNING_SECRET="+gatewayNodeSigningSecret,
		"NODE_CONTRACT_VERSION="+gatewayNodeContractVersion,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start core-platform: %v", err)
	}
	return cmd
}

func shutdownCoreProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
	case <-done:
	}
}

func waitHTTPReady(t *testing.T, readyURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(readyURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", readyURL)
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func signGatewayRegister(req map[string]any) string {
	capsAny, _ := req["capabilities"].([]string)
	caps := append([]string(nil), capsAny...)
	sort.Strings(caps)
	payload := strings.Join([]string{
		"register",
		toString(req["tenant_id"]),
		toString(req["region"]),
		toString(req["hostname"]),
		toString(req["node_key_id"]),
		toString(req["node_public_key"]),
		toString(req["contract_version"]),
		toString(req["agent_version"]),
		strings.Join(caps, ","),
		identitySerialFromMap(req["tls_identity"]),
		toString(req["nonce"]),
		strconv.FormatInt(toInt64(req["signed_at"]), 10),
	}, "\n")
	return signGatewayPayload(payload)
}

func signGatewayHeartbeat(req map[string]any, nodePublicKey string) string {
	payload := strings.Join([]string{
		"heartbeat",
		toString(req["tenant_id"]),
		toString(req["node_id"]),
		toString(req["node_key_id"]),
		nodePublicKey,
		toString(req["contract_version"]),
		toString(req["agent_version"]),
		identitySerialFromMap(req["tls_identity"]),
		toString(req["nonce"]),
		strconv.FormatInt(toInt64(req["signed_at"]), 10),
	}, "\n")
	return signGatewayPayload(payload)
}

func identitySerialFromMap(v any) string {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return ""
	}
	return strings.TrimSpace(toString(m["serial_number"]))
}

func signGatewayPayload(payload string) string {
	h := hmac.New(sha256.New, []byte(gatewayNodeSigningSecret))
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
