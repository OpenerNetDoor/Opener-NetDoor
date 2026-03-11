package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestAuditLogsInjectTenantForScopedActor(t *testing.T) {
	var gotRawQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/audit/logs" {
			gotRawQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"limit":20,"offset":0}`))
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

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/audit/logs", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotRawQuery != "tenant_id=tenant-a" {
		t.Fatalf("expected forwarded tenant_id=tenant-a, got %q", gotRawQuery)
	}
}

func TestOpsSnapshotTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/ops/snapshot?tenant_id=tenant-b", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ops snapshot tenant mismatch, got %d", rr.Code)
	}
}
func TestOpsAnalyticsTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/ops/analytics?tenant_id=tenant-b", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ops analytics tenant mismatch, got %d", rr.Code)
	}
}
