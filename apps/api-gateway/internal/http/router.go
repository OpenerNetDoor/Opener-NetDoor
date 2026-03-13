package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/handlers"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/middleware"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/response"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/proxy"
)

func NewHandler(cfg config.Config) (http.Handler, error) {
	px, err := proxy.New(cfg.CorePlatformBaseURL)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", handlers.Health)
	mux.HandleFunc("/v1/ready", handlers.Ready(px))

	authn := middleware.Authenticate(cfg)
	mux.Handle("/v1/auth/session", chain(
		http.HandlerFunc(handlers.SessionInfo()),
		authn,
	))
	mux.Handle("/v1/auth/logout", chain(
		http.HandlerFunc(handlers.Logout(cfg)),
		authn,
	))
	mux.Handle("/v1/admin/tenants", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			if r.Method == http.MethodPost && !p.IsPlatformAdmin() {
				response.Error(w, r, http.StatusForbidden, "forbidden", "tenant-scoped actor cannot create tenants")
				return
			}

			forwardReq := r
			if r.Method == http.MethodGet && p.TenantID != "" && !p.IsPlatformAdmin() {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
			}

			px.Forward(w, forwardReq, "/internal/v1/tenants", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/users", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForUsers(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/users", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/users/block", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/users/block", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/users/unblock", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/users/unblock", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/access-keys", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForAccessKeys(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if tenantID != "" && !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/access-keys", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/subscriptions/user", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/subscriptions/user", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/policies/tenants", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForTenantPolicies(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}

			forwardReq := r
			if r.Method == http.MethodGet && p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}

			if tenantID != "" && !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/policies/tenants", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/policies/users", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForUserPolicies(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if tenantID == "" {
				response.Error(w, r, http.StatusBadRequest, "validation_error", "tenant_id is required")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/policies/users", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/policies/effective", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForEffectivePolicy(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request")
				return
			}

			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}

			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/policies/effective", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/devices/register", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/devices/register", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForNodes(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}

			forwardReq := r
			if r.Method == http.MethodGet && p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}

			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/nodes", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/detail", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/nodes/detail", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))
	mux.Handle("/v1/admin/nodes/register", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/register", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/heartbeat", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/heartbeat", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/revoke", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/revoke", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/reactivate", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/reactivate", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/runtime/config", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/nodes/runtime/config", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/runtime/apply", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/runtime/apply", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))
	mux.Handle("/v1/admin/nodes/certificates", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantForNodeCertificates(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request")
				return
			}

			forwardReq := r
			if r.Method == http.MethodGet && p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}

			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/nodes/certificates", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/certificates/rotate", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/certificates/rotate", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/certificates/revoke", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/certificates/revoke", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/certificates/renew", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID, err := requestedTenantFromBody(r)
			if err != nil {
				response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
				return
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, r, "/internal/v1/nodes/certificates/renew", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/pki/issuers", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			if !p.IsPlatformAdmin() {
				response.Error(w, r, http.StatusForbidden, "forbidden", "only platform admin can manage pki issuers")
				return
			}
			px.Forward(w, r, "/internal/v1/pki/issuers", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/pki/issuers/activate", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := middleware.GetPrincipal(r.Context())
			if !p.IsPlatformAdmin() {
				response.Error(w, r, http.StatusForbidden, "forbidden", "only platform admin can manage pki issuers")
				return
			}
			px.Forward(w, r, "/internal/v1/pki/issuers/activate", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/audit/logs", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/audit/logs", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/ops/snapshot", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/ops/snapshot", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/ops/analytics", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/ops/analytics", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.Handle("/v1/admin/nodes/provisioning", chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			p, _ := middleware.GetPrincipal(r.Context())
			tenantID := r.URL.Query().Get("tenant_id")
			forwardReq := r
			if p.TenantID != "" && !p.IsPlatformAdmin() && tenantID == "" {
				q := r.URL.Query()
				q.Set("tenant_id", p.TenantID)
				cloned := r.Clone(r.Context())
				u := *r.URL
				u.RawQuery = q.Encode()
				cloned.URL = &u
				forwardReq = cloned
				tenantID = p.TenantID
			}
			if !p.CanAccessTenant(tenantID) {
				response.Error(w, r, http.StatusForbidden, "forbidden", "actor cannot access requested tenant")
				return
			}
			px.Forward(w, forwardReq, "/internal/v1/nodes/provisioning", p.Subject, p.Scopes, p.TenantID)
		}),
		authn,
		methodScope(),
	))

	mux.HandleFunc("/", handlers.MagicPath(cfg))

	h := middleware.RequestID(mux)
	return h, nil
}

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func methodScope() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := ""
			switch r.Method {
			case http.MethodGet:
				scope = "admin:read"
			case http.MethodPost, http.MethodDelete, http.MethodPut:
				scope = "admin:write"
			default:
				response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
				return
			}
			middleware.RequireScope(scope)(next).ServeHTTP(w, r)
		})
	}
}

func requestedTenantForUsers(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPost:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}

func requestedTenantForAccessKeys(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPost:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}

func requestedTenantForTenantPolicies(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPut:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}

func requestedTenantForUserPolicies(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPut:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}

func requestedTenantForEffectivePolicy(r *http.Request) (string, error) {
	if r.Method != http.MethodGet {
		return "", nil
	}
	return r.URL.Query().Get("tenant_id"), nil
}

func requestedTenantForNodes(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPost:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}
func requestedTenantForNodeCertificates(r *http.Request) (string, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query().Get("tenant_id"), nil
	case http.MethodPost:
		return requestedTenantFromBody(r)
	default:
		return "", nil
	}
}
func requestedTenantFromBody(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 {
		return "", nil
	}
	var in struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return "", err
	}
	return in.TenantID, nil
}

