package handlers

import (
	"crypto/subtle"
	"io"
	"net/http"
	"net/url"
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
	client := &http.Client{Timeout: 15 * time.Second}
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
		subjectUUID := matches[2]

		if subtle.ConstantTimeCompare([]byte(pathSecret), []byte(cfg.AdminMagicSecret)) == 1 {
			handleAdminMagicPath(w, r, cfg, subjectUUID)
			return
		}
		if subtle.ConstantTimeCompare([]byte(pathSecret), []byte(cfg.SubscriptionAccessSecret)) == 1 {
			handleSubscriptionMagicPath(w, r, client, cfg, subjectUUID)
			return
		}

		response.Error(w, r, http.StatusForbidden, "forbidden", "invalid access link")
	}
}

func handleAdminMagicPath(w http.ResponseWriter, r *http.Request, cfg config.Config, ownerUUID string) {
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

func handleSubscriptionMagicPath(w http.ResponseWriter, r *http.Request, client *http.Client, cfg config.Config, userUUID string) {
	endpoint, err := url.Parse(strings.TrimRight(cfg.CorePlatformBaseURL, "/") + "/internal/v1/subscriptions/user")
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "subscription_proxy_failed", "failed to build subscription endpoint")
		return
	}
	query := endpoint.Query()
	query.Set("tenant_id", cfg.OwnerScopeID)
	query.Set("user_id", userUUID)
	query.Set("format", "plain")
	endpoint.RawQuery = query.Encode()

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "subscription_proxy_failed", "failed to prepare subscription request")
		return
	}
	upstreamReq.Header.Set("X-Actor-Sub", cfg.OwnerSubject)
	upstreamReq.Header.Set("X-Actor-Tenant-ID", cfg.OwnerScopeID)
	upstreamReq.Header.Set("X-Actor-Scopes", "admin:read")

	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		response.Error(w, r, http.StatusBadGateway, "subscription_upstream_unavailable", "failed to resolve subscription")
		return
	}
	defer upstreamResp.Body.Close()

	body, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		response.Error(w, r, http.StatusBadGateway, "subscription_upstream_invalid", "failed to read subscription response")
		return
	}

	requestID := upstreamResp.Header.Get("X-Request-Id")
	if requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}
	contentType := upstreamResp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = w.Write(body)
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
