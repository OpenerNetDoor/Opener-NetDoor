package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/auth"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/response"
)

type principalKey struct{}

type Principal struct {
	Subject  string
	Scopes   []string
	TenantID string
}

func (p Principal) IsPlatformAdmin() bool {
	return auth.HasScope(p.Scopes, "platform:admin")
}

func (p Principal) CanAccessTenant(tenantID string) bool {
	if tenantID == "" {
		return p.IsPlatformAdmin() || p.TenantID == ""
	}
	if p.IsPlatformAdmin() || p.TenantID == "" {
		return true
	}
	return p.TenantID == tenantID
}

func GetPrincipal(ctx context.Context) (Principal, bool) {
	v, ok := ctx.Value(principalKey{}).(Principal)
	return v, ok
}

func Authenticate(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.ParseAndVerify(token, cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), principalKey{}, Principal{
				Subject:  claims.Subject,
				Scopes:   claims.Scopes,
				TenantID: claims.TenantID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := GetPrincipal(r.Context())
			if !ok {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", "principal not found")
				return
			}
			if !auth.HasScope(p.Scopes, scope) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "insufficient scope")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
