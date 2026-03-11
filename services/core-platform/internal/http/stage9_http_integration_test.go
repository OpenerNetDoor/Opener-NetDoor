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

func TestHTTPAuditLogsAndOpsSnapshotWithPostgres(t *testing.T) {
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

	tenant := createTenant(t, ts.URL, platformHeaders, testutil.UniqueName("tenant-http-stage9"))
	user := createUser(t, ts.URL, platformHeaders, tenant.ID, "http-stage9-user@example.com")

	if _, err := db.Exec(`
		INSERT INTO traffic_usage_hourly (tenant_id, user_id, protocol, ts_hour, bytes_in, bytes_out)
		VALUES ($1, $2, 'vless', $3, 120, 80)
	`, tenant.ID, user.ID, time.Now().UTC().Truncate(time.Hour)); err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	if err := s.InsertAuditLog(t.Context(), model.AuditLogEvent{
		TenantID:   tenant.ID,
		ActorType:  "node",
		ActorSub:   "node-http-stage9",
		Action:     "node.replay_rejected",
		TargetType: "node",
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("insert audit log: %v", err)
	}

	logsReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/audit/logs?tenant_id="+tenant.ID+"&limit=10", nil)
	for k, v := range platformHeaders {
		logsReq.Header.Set(k, v)
	}
	logsResp, err := http.DefaultClient.Do(logsReq)
	if err != nil {
		t.Fatalf("audit logs request: %v", err)
	}
	defer logsResp.Body.Close()
	if logsResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(logsResp.Body)
		t.Fatalf("expected 200 audit logs, got %d body=%s", logsResp.StatusCode, buf.String())
	}
	var logsPayload struct {
		Items []model.AuditLogRecord `json:"items"`
	}
	if err := json.NewDecoder(logsResp.Body).Decode(&logsPayload); err != nil {
		t.Fatalf("decode logs payload: %v", err)
	}
	if len(logsPayload.Items) == 0 {
		t.Fatal("expected at least one audit log item")
	}

	snapshotReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/ops/snapshot?tenant_id="+tenant.ID, nil)
	for k, v := range platformHeaders {
		snapshotReq.Header.Set(k, v)
	}
	snapshotResp, err := http.DefaultClient.Do(snapshotReq)
	if err != nil {
		t.Fatalf("ops snapshot request: %v", err)
	}
	defer snapshotResp.Body.Close()
	if snapshotResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(snapshotResp.Body)
		t.Fatalf("expected 200 ops snapshot, got %d body=%s", snapshotResp.StatusCode, buf.String())
	}
	var snapshot model.OpsSnapshot
	if err := json.NewDecoder(snapshotResp.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode ops snapshot: %v", err)
	}
	if snapshot.TrafficBytes24h < 200 {
		t.Fatalf("expected traffic_bytes_24h >= 200, got %d", snapshot.TrafficBytes24h)
	}
	if snapshot.ReplayRejected24h < 1 {
		t.Fatalf("expected replay_rejected_24h >= 1, got %d", snapshot.ReplayRejected24h)
	}

	analyticsReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/ops/analytics?tenant_id="+tenant.ID, nil)
	for k, v := range platformHeaders {
		analyticsReq.Header.Set(k, v)
	}
	analyticsResp, err := http.DefaultClient.Do(analyticsReq)
	if err != nil {
		t.Fatalf("ops analytics request: %v", err)
	}
	defer analyticsResp.Body.Close()
	if analyticsResp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(analyticsResp.Body)
		t.Fatalf("expected 200 ops analytics, got %d body=%s", analyticsResp.StatusCode, buf.String())
	}
	var analytics model.OpsAnalytics
	if err := json.NewDecoder(analyticsResp.Body).Decode(&analytics); err != nil {
		t.Fatalf("decode ops analytics: %v", err)
	}
	if analytics.TotalUsers < 1 {
		t.Fatalf("expected total_users >= 1, got %d", analytics.TotalUsers)
	}
	if len(analytics.ProtocolUsage24h) == 0 {
		t.Fatal("expected non-empty protocol usage")
	}

	tenantScopedHeaders := map[string]string{
		"X-Actor-Sub":       "tenant-admin",
		"X-Actor-Tenant-ID": tenant.ID,
		"X-Actor-Scopes":    "admin:read",
	}
	denyReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/internal/v1/audit/logs?tenant_id="+testutil.UniqueName("foreign"), nil)
	for k, v := range tenantScopedHeaders {
		denyReq.Header.Set(k, v)
	}
	denyResp, err := http.DefaultClient.Do(denyReq)
	if err != nil {
		t.Fatalf("deny audit logs request: %v", err)
	}
	defer denyResp.Body.Close()
	if denyResp.StatusCode != http.StatusForbidden {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(denyResp.Body)
		t.Fatalf("expected 403 for foreign tenant audit logs, got %d body=%s", denyResp.StatusCode, buf.String())
	}
}
