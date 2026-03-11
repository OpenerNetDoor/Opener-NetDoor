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

func TestUserBlockTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/users/block", bytes.NewBufferString(`{"tenant_id":"tenant-b","user_id":"u1"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user block tenant mismatch, got %d", rr.Code)
	}
}

func TestUserDeleteTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/admin/users?id=u1&tenant_id=tenant-b", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for user delete tenant mismatch, got %d", rr.Code)
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

func TestNodeCreateTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/nodes", bytes.NewBufferString(`{"tenant_id":"tenant-b","region":"de","hostname":"de-1.example.com"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for node create tenant mismatch, got %d", rr.Code)
	}
}

func TestNodeDetailInjectsTenantForScopedActor(t *testing.T) {
	var gotRawQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/nodes/detail" {
			gotRawQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"n1","tenant_id":"tenant-a","region":"de","hostname":"de-1.example.com","node_key_id":"nk1","node_public_key":"pk","contract_version":"2026-03-10.stage5.v1","agent_version":"owner-manual","capabilities":[],"identity_fingerprint":"fp","status":"pending","created_at":"2026-03-11T00:00:00Z"}`))
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
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/nodes/detail?node_id=n1", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotRawQuery != "node_id=n1&tenant_id=tenant-a" && gotRawQuery != "tenant_id=tenant-a&node_id=n1" {
		t.Fatalf("expected tenant injection in forwarded query, got %q", gotRawQuery)
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

func TestNodeRevokeTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/nodes/revoke", bytes.NewBufferString(`{"tenant_id":"tenant-b","node_id":"node-1"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for node revoke tenant mismatch, got %d", rr.Code)
	}
}

func TestNodeReactivateTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/nodes/reactivate", bytes.NewBufferString(`{"tenant_id":"tenant-b","node_id":"node-1"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for node reactivate tenant mismatch, got %d", rr.Code)
	}
}

func TestNodeCertificatesTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/nodes/certificates/rotate", bytes.NewBufferString(`{"tenant_id":"tenant-b","node_id":"node-1"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for node certificate rotate tenant mismatch, got %d", rr.Code)
	}
}

func TestNodeCertificateRenewTenantIsolationDenyPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/nodes/certificates/renew", bytes.NewBufferString(`{"tenant_id":"tenant-b","node_id":"node-1"}`))
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:write"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for node certificate renew tenant mismatch, got %d", rr.Code)
	}
}

func TestPKIIssuersPlatformAdminOnly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()

	cfg := config.Config{HTTPAddr: ":8080", CorePlatformBaseURL: upstream.URL, JWTIssuer: "iss", JWTAudience: "aud", JWTSecret: "very-secure-secret"}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/pki/issuers", nil)
	req.Header.Set("Authorization", "Bearer "+testutil.MustIssueToken(t, testutil.TokenParams{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, Scopes: []string{"admin:read"}, TenantID: "tenant-a"}))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-platform pki issuer list, got %d", rr.Code)
	}
}
