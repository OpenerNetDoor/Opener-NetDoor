package handlers

import (
	"crypto/subtle"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/middleware"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/response"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/session"
)

var magicPathRe = regexp.MustCompile(`^/([A-Za-z0-9_-]{16,})/([0-9a-fA-F-]{36})/?$`)

func MagicPath(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
			return
		}

		matches := magicPathRe.FindStringSubmatch(r.URL.Path)
		if len(matches) != 3 {
			response.Error(w, r, http.StatusNotFound, "not_found", "route not found")
			return
		}

		pathSecret := matches[1]
		ownerUUID := matches[2]
		if subtle.ConstantTimeCompare([]byte(pathSecret), []byte(cfg.AdminMagicSecret)) != 1 {
			response.Error(w, r, http.StatusForbidden, "forbidden", "invalid admin access link")
			return
		}
		if !strings.EqualFold(ownerUUID, cfg.OwnerScopeID) {
			response.Error(w, r, http.StatusForbidden, "forbidden", "invalid owner scope")
			return
		}

		token, expiresAt, err := session.Issue(
			cfg.SessionSecret,
			cfg.JWTIssuer,
			cfg.JWTAudience,
			cfg.SessionTTL,
			cfg.OwnerSubject,
			cfg.OwnerScopeID,
			[]string{"admin:read", "admin:write", "platform:admin"},
		)
		if err != nil {
			response.Error(w, r, http.StatusInternalServerError, "session_issue_failed", "failed to create session")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cfg.SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.SessionSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  expiresAt,
			MaxAge:   int(time.Until(expiresAt).Seconds()),
		})

		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func SessionInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := middleware.GetPrincipal(r.Context())
		if !ok {
			response.Error(w, r, http.StatusUnauthorized, "unauthorized", "principal not found")
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"subject":       p.Subject,
			"tenant_id":     p.TenantID,
			"scopes":        p.Scopes,
			"expires_at":    time.Unix(p.Exp, 0).UTC(),
		})
	}
}

func Logout(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cfg.SessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.SessionSecure,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})

		response.JSON(w, http.StatusOK, map[string]any{"logged_out": true})
	}
}
