package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestReadyEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/ready" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?tenant_id=tenant-b", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestAccessKeysTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/access-keys", bytes.NewBufferString(`{"tenant_id":"tenant-b","user_id":"u1","key_type":"vless"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for tenant mismatch, got %d", rr.Code)
	}
}

func TestPolicyTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/admin/policies/tenants", bytes.NewBufferString(`{"tenant_id":"tenant-b","traffic_quota_bytes":1024}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for tenant policy mismatch, got %d", rr.Code)
	}
}

func TestPolicyTenantListInjectsTenantForScopedActor(t *testing.T) {
	var gotRawQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/policies/tenants" {
			gotRawQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"limit":20,"offset":0}`))
			return
		}
		if r.URL.Path == "/internal/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/policies/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotRawQuery != "tenant_id=tenant-a" {
		t.Fatalf("expected forwarded tenant_id=tenant-a, got %q", gotRawQuery)
	}
}

func TestNodesListInjectsTenantForScopedActor(t *testing.T) {
	var gotRawQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/nodes" {
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
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotRawQuery != "tenant_id=tenant-a" {
		t.Fatalf("expected forwarded tenant_id=tenant-a, got %q", gotRawQuery)
	}
}

func TestRequestIDNotDuplicatedFromUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/v1/users", "/internal/v1/access-keys", "/internal/v1/policies/tenants":
			w.Header().Set("X-Request-ID", "upstream-rid")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[]}`))
		case "/internal/ready":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/policies/tenants?tenant_id=tenant-a", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	values := rr.Header()["X-Request-Id"]
	if len(values) != 1 {
		t.Fatalf("expected exactly one X-Request-Id, got %v", values)
	}
	if values[0] == "upstream-rid" {
		t.Fatalf("expected gateway-generated request id, got upstream value %q", values[0])
	}
}
