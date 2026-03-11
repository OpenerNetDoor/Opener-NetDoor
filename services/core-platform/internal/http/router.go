package http

import (
	"net/http"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/http/handlers"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/http/middleware"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/service"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
)

type Options struct {
	ServiceOptions service.Options
}

func NewHandler(s store.Store, opts ...Options) http.Handler {
	cfg := Options{}
	if len(opts) > 0 {
		cfg = opts[0]
	}
	svc := service.New(s, cfg.ServiceOptions)
	return NewHandlerWithService(svc)
}

func NewHandlerWithService(svc service.Service) http.Handler {
	h := handlers.New(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/health", h.Health)
	mux.HandleFunc("/internal/ready", h.Ready)
	mux.HandleFunc("/internal/v1/tenants", h.Tenants)
	mux.HandleFunc("/internal/v1/users", h.Users)
	mux.HandleFunc("/internal/v1/users/block", h.UsersBlock)
	mux.HandleFunc("/internal/v1/users/unblock", h.UsersUnblock)
	mux.HandleFunc("/internal/v1/access-keys", h.AccessKeys)
	mux.HandleFunc("/internal/v1/policies/tenants", h.TenantPolicies)
	mux.HandleFunc("/internal/v1/policies/users", h.UserPolicies)
	mux.HandleFunc("/internal/v1/policies/effective", h.EffectivePolicy)
	mux.HandleFunc("/internal/v1/devices/register", h.Devices)
	mux.HandleFunc("/internal/v1/nodes", h.Nodes)
	mux.HandleFunc("/internal/v1/nodes/detail", h.NodeDetail)
	mux.HandleFunc("/internal/v1/nodes/register", h.NodeRegister)
	mux.HandleFunc("/internal/v1/nodes/heartbeat", h.NodeHeartbeat)
	mux.HandleFunc("/internal/v1/nodes/revoke", h.NodeRevoke)
	mux.HandleFunc("/internal/v1/nodes/reactivate", h.NodeReactivate)
	mux.HandleFunc("/internal/v1/nodes/certificates", h.NodeCertificates)
	mux.HandleFunc("/internal/v1/nodes/certificates/rotate", h.NodeCertificatesRotate)
	mux.HandleFunc("/internal/v1/nodes/certificates/revoke", h.NodeCertificatesRevoke)
	mux.HandleFunc("/internal/v1/nodes/certificates/renew", h.NodeCertificatesRenew)
	mux.HandleFunc("/internal/v1/pki/issuers", h.PKIIssuers)
	mux.HandleFunc("/internal/v1/pki/issuers/activate", h.PKIIssuersActivate)
	mux.HandleFunc("/internal/v1/audit/logs", h.AuditLogs)
	mux.HandleFunc("/internal/v1/ops/snapshot", h.OpsSnapshot)
	mux.HandleFunc("/internal/v1/nodes/provisioning", h.NodeProvisioning)
	return middleware.RequestID(mux)
}
