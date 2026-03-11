//go:build integration

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestGatewayAuditLogsAndOpsSnapshotWithLiveCore(t *testing.T) {
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

	platformToken := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read", "admin:write", "platform:admin"},
	})
	platformHeaders := map[string]string{"Authorization": "Bearer " + platformToken}

	tenant := gatewayCreateTenant(t, gw.URL, platformHeaders, uniqueName("tenant-gw-stage9"))
	user := gatewayCreateUser(t, gw.URL, platformHeaders, tenant.ID, "gw-stage9-user@example.com")

	if _, err := db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 111, 222)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour)); err != nil {
		t.Fatalf("insert usage: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO audit_logs (tenant_id, actor_type, actor_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1::uuid, 'node', NULL, 'node.replay_rejected', 'node', NULL, '{}'::jsonb, NOW())
	`, tenant.ID); err != nil {
		t.Fatalf("insert audit log: %v", err)
	}

	status, body := gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/audit/logs?tenant_id="+tenant.ID+"&limit=10", platformHeaders, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 audit logs, got %d body=%s", status, body)
	}
	var logsPayload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &logsPayload); err != nil {
		t.Fatalf("decode audit logs payload: %v", err)
	}
	if len(logsPayload.Items) == 0 {
		t.Fatal("expected non-empty audit logs from gateway")
	}

	status, body = gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/ops/snapshot?tenant_id="+tenant.ID, platformHeaders, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 ops snapshot, got %d body=%s", status, body)
	}
	var snapshot struct {
		TrafficBytes24h   int64 `json:"traffic_bytes_24h"`
		ReplayRejected24h int   `json:"replay_rejected_24h"`
	}
	if err := json.Unmarshal([]byte(body), &snapshot); err != nil {
		t.Fatalf("decode ops snapshot payload: %v", err)
	}
	if snapshot.TrafficBytes24h < 333 {
		t.Fatalf("expected traffic bytes >= 333, got %d", snapshot.TrafficBytes24h)
	}
	if snapshot.ReplayRejected24h < 1 {
		t.Fatalf("expected replay rejected >= 1, got %d", snapshot.ReplayRejected24h)
	}

	tenantToken := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read"},
		TenantID: tenant.ID,
	})
	tenantHeaders := map[string]string{"Authorization": "Bearer " + tenantToken}
	status, body = gatewayRequest(t, http.MethodGet, gw.URL+"/v1/admin/audit/logs?tenant_id="+uniqueName("foreign-tenant"), tenantHeaders, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign tenant audit logs, got %d body=%s", status, body)
	}
}

func TestGatewayAuditLogsPassesThroughSinceQuery(t *testing.T) {
	gotQuery := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/audit/logs" {
			gotQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	server := httptest.NewServer(h)
	defer server.Close()

	token := testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read", "platform:admin"}})
	headers := map[string]string{"Authorization": "Bearer " + token}

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/admin/audit/logs?since=not-a-time", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 pass-through, got %d body=%s", resp.StatusCode, buf.String())
	}
	if gotQuery != "since=not-a-time" {
		t.Fatalf("expected forwarded query since=not-a-time, got %q", gotQuery)
	}
}
