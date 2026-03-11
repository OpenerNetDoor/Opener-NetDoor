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
	Exp      int64
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
			principal, err := principalFromRequest(r, cfg)
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "unauthorized", err.Error())
				return
			}
			ctx := context.WithValue(r.Context(), principalKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func principalFromRequest(r *http.Request, cfg config.Config) (Principal, error) {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseAndVerify(token, cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience)
		if err != nil {
			return Principal{}, err
		}
		return Principal{
			Subject:  claims.Subject,
			Scopes:   claims.Scopes,
			TenantID: claims.TenantID,
			Exp:      claims.ExpiresAt.Unix(),
		}, nil
	}

	cookie, err := r.Cookie(cfg.SessionCookieName)
	if err != nil {
		return Principal{}, err
	}
	claims, err := auth.ParseAndVerify(cookie.Value, cfg.SessionSecret, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		return Principal{}, err
	}
	return Principal{
		Subject:  claims.Subject,
		Scopes:   claims.Scopes,
		TenantID: claims.TenantID,
		Exp:      claims.ExpiresAt.Unix(),
	}, nil
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
