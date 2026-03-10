//go:build integration

package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/testutil"
)

const (
	testNodeSigningSecret   = "opener-netdoor-stage5-dev-signing-secret"
	testNodeContractVersion = "2026-03-10.stage5.v1"
)

func TestHTTPAccessKeysLifecycleWithPostgres(t *testing.T) {
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

	tenant := createTenant(t, ts.URL, actorHeaders, testutil.UniqueName("tenant-http"))
	user := createUser(t, ts.URL, actorHeaders, tenant.ID, "http-user@example.com")

	created := createAccessKey(t, ts.URL, actorHeaders, model.CreateAccessKeyRequest{TenantID: tenant.ID, UserID: user.ID, KeyType: "vless"})
	if created.Status != "active" {
		t.Fatalf("expected active access key, got %s", created.Status)
	}

	listURL := ts.URL + "/internal/v1/access-keys?tenant_id=" + tenant.ID + "&user_id=" + user.ID + "&limit=10&offset=0"
	req, _ := http.NewRequest(http.MethodGet, listURL, nil)
	for k, v := range actorHeaders {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list access keys request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var listed struct {
		Items []model.AccessKey `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != created.ID {
		t.Fatalf("expected created key in list, got %+v", listed.Items)
	}

	revokeReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/internal/v1/access-keys?id="+created.ID+"&tenant_id="+tenant.ID, nil)
	for k, v := range actorHeaders {
		revokeReq.Header.Set(k, v)
	}
	revokeResp, err := http.DefaultClient.Do(revokeReq)
	if err != nil {
		t.Fatalf("revoke request: %v", err)
	}
	defer revokeResp.Body.Close()
	if revokeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 revoke, got %d", revokeResp.StatusCode)
	}
	var revoked model.AccessKey
	if err := json.NewDecoder(revokeResp.Body).Decode(&revoked); err != nil {
		t.Fatalf("decode revoke response: %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("expected revoked status, got %s", revoked.Status)
	}
}

func TestHTTPPolicyQuotaAndDeviceEnforcementWithPostgres(t *testing.T) {
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

	tenant := createTenant(t, ts.URL, actorHeaders, testutil.UniqueName("tenant-http-policy"))
	user := createUser(t, ts.URL, actorHeaders, tenant.ID, "http-policy-user@example.com")

	quota := int64(1000)
	devLimit := 1
	ttl := 120
	setTenantPolicy(t, ts.URL, actorHeaders, model.SetTenantPolicyRequest{
		TenantID:             tenant.ID,
		TrafficQuotaBytes:    &quota,
		DeviceLimit:          &devLimit,
		DefaultKeyTTLSeconds: &ttl,
	})

	effective := getEffectivePolicy(t, ts.URL, actorHeaders, tenant.ID, user.ID)
	if effective.DeviceLimit == nil || *effective.DeviceLimit != 1 {
		t.Fatalf("expected effective device limit 1, got %+v", effective.DeviceLimit)
	}

	registerDevice(t, ts.URL, actorHeaders, model.RegisterDeviceRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		DeviceFingerprint: "fp-http-1",
		Platform:          "ios",
	})

	status, body := registerDeviceExpect(t, ts.URL, actorHeaders, model.RegisterDeviceRequest{
		TenantID:          tenant.ID,
		UserID:            user.ID,
		DeviceFingerprint: "fp-http-2",
		Platform:          "android",
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

	status, body = createAccessKeyExpect(t, ts.URL, actorHeaders, model.CreateAccessKeyRequest{TenantID: tenant.ID, UserID: user.ID, KeyType: "vless"})
	if status != http.StatusConflict {
		t.Fatalf("expected 409 for quota exceeded, got %d body=%s", status, body)
	}
}

func TestHTTPNodeRegistrationHeartbeatWithPostgres(t *testing.T) {
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

	tenant := createTenant(t, ts.URL, actorHeaders, testutil.UniqueName("tenant-http-node"))
	signedAt := time.Now().UTC().Unix()
	caps := []string{"heartbeat.v1", "provisioning.v1"}
	registerReq := model.RegisterNodeRequest{
		TenantID:        tenant.ID,
		Region:          "eu-central",
		Hostname:        "node-http-1",
		NodeKeyID:       "node-key-http-1",
		NodePublicKey:   "pubkey-http-1",
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.2.3",
		Capabilities:    caps,
		SignedAt:        signedAt,
	}
	registerReq.Signature = signRegisterNode(registerReq)

	registration := registerNode(t, ts.URL, actorHeaders, registerReq)
	if registration.Node.Status == "" {
		t.Fatal("expected node status in registration response")
	}
	if registration.Provisioning.ContractVersion != testNodeContractVersion {
		t.Fatalf("unexpected contract version: %s", registration.Provisioning.ContractVersion)
	}

	heartbeatReq := model.NodeHeartbeatRequest{
		TenantID:        tenant.ID,
		NodeID:          registration.Node.ID,
		NodeKeyID:       registration.Node.NodeKeyID,
		ContractVersion: testNodeContractVersion,
		AgentVersion:    "1.2.3",
		SignedAt:        time.Now().UTC().Unix(),
	}
	heartbeatReq.Signature = signHeartbeatNode(heartbeatReq, registerReq.NodePublicKey)
	node := heartbeatNode(t, ts.URL, actorHeaders, heartbeatReq)
	if node.Status != "active" {
		t.Fatalf("expected active node after heartbeat, got %s", node.Status)
	}

	nodes := listNodes(t, ts.URL, actorHeaders, tenant.ID)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	provisioning := getNodeProvisioning(t, ts.URL, actorHeaders, tenant.ID, registration.Node.ID)
	if provisioning.NodeID != registration.Node.ID {
		t.Fatalf("expected provisioning for node %s, got %s", registration.Node.ID, provisioning.NodeID)
	}
}

func createTenant(t *testing.T, baseURL string, headers map[string]string, name string) model.Tenant {
	t.Helper()
	body, _ := json.Marshal(model.CreateTenantRequest{Name: name})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create tenant request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create tenant, got %d", resp.StatusCode)
	}
	var out model.Tenant
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode tenant: %v", err)
	}
	return out
}

func createUser(t *testing.T, baseURL string, headers map[string]string, tenantID string, email string) model.User {
	t.Helper()
	body, _ := json.Marshal(model.CreateUserRequest{TenantID: tenantID, Email: email})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create user request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create user, got %d", resp.StatusCode)
	}
	var out model.User
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	return out
}

func createAccessKey(t *testing.T, baseURL string, headers map[string]string, in model.CreateAccessKeyRequest) model.AccessKey {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/access-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create access key request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create access key, got %d", resp.StatusCode)
	}
	var out model.AccessKey
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode access key: %v", err)
	}
	return out
}

func createAccessKeyExpect(t *testing.T, baseURL string, headers map[string]string, in model.CreateAccessKeyRequest) (int, string) {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/access-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create access key request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String()
}

func setTenantPolicy(t *testing.T, baseURL string, headers map[string]string, in model.SetTenantPolicyRequest) {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPut, baseURL+"/internal/v1/policies/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("set tenant policy request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 set tenant policy, got %d body=%s", resp.StatusCode, buf.String())
	}
}

func getEffectivePolicy(t *testing.T, baseURL string, headers map[string]string, tenantID string, userID string) model.EffectivePolicy {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/internal/v1/policies/effective?tenant_id="+tenantID+"&user_id="+userID, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get effective policy request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 effective policy, got %d", resp.StatusCode)
	}
	var out model.EffectivePolicy
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode effective policy: %v", err)
	}
	return out
}

func registerDevice(t *testing.T, baseURL string, headers map[string]string, in model.RegisterDeviceRequest) model.Device {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/devices/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register device request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 201 register device, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.Device
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode device: %v", err)
	}
	return out
}

func registerDeviceExpect(t *testing.T, baseURL string, headers map[string]string, in model.RegisterDeviceRequest) (int, string) {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/devices/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register device request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String()
}

func registerNode(t *testing.T, baseURL string, headers map[string]string, in model.RegisterNodeRequest) model.NodeRegistrationResult {
	t.Helper()
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/internal/v1/nodes/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register node request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 201 register node, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.NodeRegistrationResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode node registration: %v", err)
	}
	return out
}

func heartbeatNode(t *testing.T, baseURL string, headers map[string]string, in model.NodeHeartbeatRequest) model.Node {
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
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 node heartbeat, got %d body=%s", resp.StatusCode, buf.String())
	}
	var out model.Node
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode node heartbeat: %v", err)
	}
	return out
}

func listNodes(t *testing.T, baseURL string, headers map[string]string, tenantID string) []model.Node {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/internal/v1/nodes?tenant_id="+tenantID, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list nodes request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 list nodes, got %d", resp.StatusCode)
	}
	var out struct {
		Items []model.Node `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode list nodes: %v", err)
	}
	return out.Items
}

func getNodeProvisioning(t *testing.T, baseURL string, headers map[string]string, tenantID string, nodeID string) model.NodeProvisioningContract {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/internal/v1/nodes/provisioning?tenant_id="+tenantID+"&node_id="+nodeID, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("node provisioning request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 node provisioning, got %d", resp.StatusCode)
	}
	var out model.NodeProvisioningContract
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode node provisioning: %v", err)
	}
	return out
}

func signRegisterNode(in model.RegisterNodeRequest) string {
	caps := append([]string(nil), in.Capabilities...)
	sort.Strings(caps)
	payload := strings.Join([]string{
		"register",
		in.TenantID,
		in.Region,
		in.Hostname,
		in.NodeKeyID,
		in.NodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		strings.Join(caps, ","),
		strconv.FormatInt(in.SignedAt, 10),
	}, "\n")
	return hmacHex(payload)
}

func signHeartbeatNode(in model.NodeHeartbeatRequest, nodePublicKey string) string {
	payload := strings.Join([]string{
		"heartbeat",
		in.TenantID,
		in.NodeID,
		in.NodeKeyID,
		nodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		strconv.FormatInt(in.SignedAt, 10),
	}, "\n")
	return hmacHex(payload)
}

func hmacHex(payload string) string {
	h := hmac.New(sha256.New, []byte(testNodeSigningSecret))
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
