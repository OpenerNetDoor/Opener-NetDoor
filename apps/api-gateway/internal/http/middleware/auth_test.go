package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/testutil"
)

func TestAuthenticateRejectsMissingToken(t *testing.T) {
	cfg := config.Config{JWTSecret: "very-secure-secret", JWTIssuer: "iss", JWTAudience: "aud"}
	h := Authenticate(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/tenants", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticateAndScopeHappyPath(t *testing.T) {
	cfg := config.Config{JWTSecret: "very-secure-secret", JWTIssuer: "iss", JWTAudience: "aud"}
	token := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read"},
		TenantID: "tenant-a",
	})

	protected := Authenticate(cfg)(RequireScope("admin:read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRequireScopeDenyPath(t *testing.T) {
	cfg := config.Config{JWTSecret: "very-secure-secret", JWTIssuer: "iss", JWTAudience: "aud"}
	token := testutil.MustIssueToken(t, testutil.TokenParams{
		Secret:   cfg.JWTSecret,
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
		Scopes:   []string{"admin:read"},
		TenantID: "tenant-a",
	})

	protected := Authenticate(cfg)(RequireScope("admin:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
