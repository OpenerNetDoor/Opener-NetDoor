package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/http/response"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/service"
)

type Handler struct {
	svc service.Service
}

type nodeOwnerService interface {
	CreateNode(ctx context.Context, actor model.ActorPrincipal, in model.CreateNodeRequest) (model.Node, error)
	GetNode(ctx context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.Node, error)
}

type runtimeOwnerService interface {
	GetNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.RuntimeConfigResponse, error)
	ApplyNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, in model.RuntimeApplyRequest) (model.RuntimeConfigResponse, error)
}

func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Health(r.Context()); err != nil {
		status, code, message := service.ToResponse(err, "health_failed", "health check failed")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "opener-netdoor-core-platform"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Health(r.Context()); err != nil {
		status, code, message := service.ToResponse(err, "ready_failed", "readiness check failed")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "opener-netdoor-core-platform"})
}

func (h *Handler) Tenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTenants(w, r)
	case http.MethodPost:
		h.createTenant(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	case http.MethodDelete:
		h.deleteUser(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) UsersBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	h.userStatusMutation(w, r, "blocked")
}

func (h *Handler) UsersUnblock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	h.userStatusMutation(w, r, "active")
}

func (h *Handler) AccessKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAccessKeys(w, r)
	case http.MethodPost:
		h.createAccessKey(w, r)
	case http.MethodDelete:
		h.revokeAccessKey(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) UserSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	item, err := h.svc.GetUserSubscription(r.Context(), actor, model.GetUserSubscriptionQuery{
		TenantID: r.URL.Query().Get("tenant_id"),
		UserID:   r.URL.Query().Get("user_id"),
		Format:   r.URL.Query().Get("format"),
	})
	if err != nil {
		status, code, message := service.ToResponse(err, "subscription_resolve_failed", "failed to resolve user subscription")
		response.Error(w, r, status, code, message)
		return
	}
	if item.Format == "plain" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(item.Payload))
		return
	}
	response.JSON(w, http.StatusOK, item)
}
func (h *Handler) TenantPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getOrListTenantPolicies(w, r)
	case http.MethodPut:
		h.setTenantPolicy(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) UserPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getOrListUserPolicies(w, r)
	case http.MethodPut:
		h.setUserPolicy(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) EffectivePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	item, err := h.svc.GetEffectivePolicy(r.Context(), actor, model.GetEffectivePolicyQuery{
		TenantID: r.URL.Query().Get("tenant_id"),
		UserID:   r.URL.Query().Get("user_id"),
	})
	if err != nil {
		status, code, message := service.ToResponse(err, "effective_policy_failed", "failed to resolve effective policy")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) Devices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RegisterDevice(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "device_register_failed", "failed to register device")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) Nodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listNodes(w, r)
	case http.MethodPost:
		h.createNode(w, r)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) NodeDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	nodeSvc, ok := h.svc.(nodeOwnerService)
	if !ok {
		response.Error(w, r, http.StatusNotImplemented, "not_implemented", "node detail is not supported by configured service")
		return
	}
	item, err := nodeSvc.GetNode(r.Context(), actor, r.URL.Query().Get("tenant_id"), r.URL.Query().Get("node_id"))
	if err != nil {
		status, code, message := service.ToResponse(err, "node_get_failed", "failed to get node")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	runtimeSvc, ok := h.svc.(runtimeOwnerService)
	if !ok {
		response.Error(w, r, http.StatusNotImplemented, "not_implemented", "node runtime is not supported by configured service")
		return
	}
	item, err := runtimeSvc.GetNodeRuntimeConfig(r.Context(), actor, r.URL.Query().Get("tenant_id"), r.URL.Query().Get("node_id"))
	if err != nil {
		status, code, message := service.ToResponse(err, "node_runtime_config_failed", "failed to generate node runtime config")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeRuntimeApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	runtimeSvc, ok := h.svc.(runtimeOwnerService)
	if !ok {
		response.Error(w, r, http.StatusNotImplemented, "not_implemented", "node runtime is not supported by configured service")
		return
	}
	var req model.RuntimeApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := runtimeSvc.ApplyNodeRuntimeConfig(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_runtime_apply_failed", "failed to apply node runtime config")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}
func (h *Handler) NodeRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RegisterNode(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_register_failed", "failed to register node")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) NodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.NodeHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.NodeHeartbeat(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_heartbeat_failed", "failed to process node heartbeat")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.NodeLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RevokeNode(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_revoke_failed", "failed to revoke node")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeReactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.NodeLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.ReactivateNode(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_reactivate_failed", "failed to reactivate node")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeCertificates(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	switch r.Method {
	case http.MethodGet:
		q := model.ListNodeCertificatesQuery{
			ListQuery: model.ListQuery{
				Limit:  parseLimit(r.URL.Query().Get("limit")),
				Offset: parseOffset(r.URL.Query().Get("offset")),
				Status: r.URL.Query().Get("status"),
			},
			TenantID: r.URL.Query().Get("tenant_id"),
			NodeID:   r.URL.Query().Get("node_id"),
		}
		items, err := h.svc.ListNodeCertificates(r.Context(), actor, q)
		if err != nil {
			status, code, message := service.ToResponse(err, "node_certificate_list_failed", "failed to list node certificates")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
	case http.MethodPost:
		var req model.RotateNodeCertificateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
			return
		}
		item, err := h.svc.IssueNodeCertificate(r.Context(), actor, req)
		if err != nil {
			status, code, message := service.ToResponse(err, "node_certificate_issue_failed", "failed to issue node certificate")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusOK, item)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) NodeCertificatesRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.RotateNodeCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RotateNodeCertificate(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_certificate_rotate_failed", "failed to rotate node certificate")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeCertificatesRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.RevokeNodeCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RevokeNodeCertificate(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_certificate_revoke_failed", "failed to revoke node certificate")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) NodeCertificatesRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.RenewNodeCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.RenewNodeCertificate(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_certificate_renew_failed", "failed to renew node certificate")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) PKIIssuers(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	switch r.Method {
	case http.MethodGet:
		q := model.ListPKIIssuersQuery{
			ListQuery: model.ListQuery{
				Limit:  parseLimit(r.URL.Query().Get("limit")),
				Offset: parseOffset(r.URL.Query().Get("offset")),
				Status: r.URL.Query().Get("status"),
			},
			Source: r.URL.Query().Get("source"),
		}
		items, err := h.svc.ListPKIIssuers(r.Context(), actor, q)
		if err != nil {
			status, code, message := service.ToResponse(err, "issuer_list_failed", "failed to list pki issuers")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
	case http.MethodPost:
		var req model.CreatePKIIssuerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
			return
		}
		item, err := h.svc.CreatePKIIssuer(r.Context(), actor, req)
		if err != nil {
			status, code, message := service.ToResponse(err, "issuer_create_failed", "failed to create pki issuer")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusCreated, item)
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
}

func (h *Handler) PKIIssuersActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	var req model.ActivatePKIIssuerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.ActivatePKIIssuer(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "issuer_activate_failed", "failed to activate pki issuer")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}
func (h *Handler) NodeProvisioning(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	item, err := h.svc.GetNodeProvisioning(r.Context(), actor, model.GetNodeProvisioningQuery{
		TenantID:  r.URL.Query().Get("tenant_id"),
		NodeID:    r.URL.Query().Get("node_id"),
		NodeKeyID: r.URL.Query().Get("node_key_id"),
	})
	if err != nil {
		status, code, message := service.ToResponse(err, "node_provisioning_failed", "failed to get node provisioning")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	since, err := parseTimeQuery(r.URL.Query().Get("since"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "validation_error", "invalid since value")
		return
	}
	until, err := parseTimeQuery(r.URL.Query().Get("until"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "validation_error", "invalid until value")
		return
	}
	q := model.ListAuditLogsQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
		},
		TenantID:   r.URL.Query().Get("tenant_id"),
		Action:     r.URL.Query().Get("action"),
		ActorType:  r.URL.Query().Get("actor_type"),
		TargetType: r.URL.Query().Get("target_type"),
		Since:      since,
		Until:      until,
	}
	items, err := h.svc.ListAuditLogs(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "audit_log_list_failed", "failed to list audit logs")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) OpsSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	item, err := h.svc.GetOpsSnapshot(r.Context(), actor, r.URL.Query().Get("tenant_id"))
	if err != nil {
		status, code, message := service.ToResponse(err, "ops_snapshot_failed", "failed to load operations snapshot")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) OpsAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
	actor := actorFromHeaders(r)
	item, err := h.svc.GetOpsAnalytics(r.Context(), actor, r.URL.Query().Get("tenant_id"))
	if err != nil {
		status, code, message := service.ToResponse(err, "ops_analytics_failed", "failed to load operations analytics")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) listTenants(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	q := model.ListQuery{
		Limit:  parseLimit(r.URL.Query().Get("limit")),
		Offset: parseOffset(r.URL.Query().Get("offset")),
		Status: r.URL.Query().Get("status"),
	}
	items, err := h.svc.ListTenants(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "tenant_list_failed", "failed to list tenants")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) createTenant(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	var req model.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.CreateTenant(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "tenant_create_failed", "failed to create tenant")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	q := model.ListUsersQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
			Status: r.URL.Query().Get("status"),
		},
		TenantID: r.URL.Query().Get("tenant_id"),
	}
	items, err := h.svc.ListUsers(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "user_list_failed", "failed to list users")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.CreateUser(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "user_create_failed", "failed to create user")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) listNodes(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	q := model.ListNodesQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
		},
		TenantID: r.URL.Query().Get("tenant_id"),
		Status:   r.URL.Query().Get("status"),
	}
	items, err := h.svc.ListNodes(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_list_failed", "failed to list nodes")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) createNode(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	nodeSvc, ok := h.svc.(nodeOwnerService)
	if !ok {
		response.Error(w, r, http.StatusNotImplemented, "not_implemented", "node create is not supported by configured service")
		return
	}
	var req model.CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := nodeSvc.CreateNode(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "node_create_failed", "failed to create node")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}
func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	req := model.UserLifecycleRequest{
		TenantID: r.URL.Query().Get("tenant_id"),
		UserID:   r.URL.Query().Get("id"),
	}
	if err := h.svc.DeleteUser(r.Context(), actor, req); err != nil {
		status, code, message := service.ToResponse(err, "user_delete_failed", "failed to delete user")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"deleted": true, "id": req.UserID, "tenant_id": req.TenantID})
}

func (h *Handler) userStatusMutation(w http.ResponseWriter, r *http.Request, status string) {
	actor := actorFromHeaders(r)
	var req model.UserLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	var (
		item model.User
		err  error
	)
	switch status {
	case "blocked":
		item, err = h.svc.BlockUser(r.Context(), actor, req)
	case "active":
		item, err = h.svc.UnblockUser(r.Context(), actor, req)
	default:
		response.Error(w, r, http.StatusBadRequest, "validation_error", "invalid status transition")
		return
	}
	if err != nil {
		fallback := "user_update_failed"
		message := "failed to update user"
		if status == "blocked" {
			fallback = "user_block_failed"
			message = "failed to block user"
		} else if status == "active" {
			fallback = "user_unblock_failed"
			message = "failed to unblock user"
		}
		statusCode, code, msg := service.ToResponse(err, fallback, message)
		response.Error(w, r, statusCode, code, msg)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) listAccessKeys(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	q := model.ListAccessKeysQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
			Status: r.URL.Query().Get("status"),
		},
		TenantID: r.URL.Query().Get("tenant_id"),
		UserID:   r.URL.Query().Get("user_id"),
	}
	items, err := h.svc.ListAccessKeys(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "access_key_list_failed", "failed to list access keys")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) createAccessKey(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	var req model.CreateAccessKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.CreateAccessKey(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "access_key_create_failed", "failed to create access key")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *Handler) revokeAccessKey(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	req := model.RevokeAccessKeyRequest{
		ID:       r.URL.Query().Get("id"),
		TenantID: r.URL.Query().Get("tenant_id"),
	}
	item, err := h.svc.RevokeAccessKey(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "access_key_revoke_failed", "failed to revoke access key")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) getOrListTenantPolicies(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID != "" {
		item, err := h.svc.GetTenantPolicy(r.Context(), actor, tenantID)
		if err != nil {
			status, code, message := service.ToResponse(err, "tenant_policy_get_failed", "failed to get tenant policy")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusOK, item)
		return
	}

	q := model.ListTenantPoliciesQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
		},
		TenantID: tenantID,
	}
	items, err := h.svc.ListTenantPolicies(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "tenant_policy_list_failed", "failed to list tenant policies")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) setTenantPolicy(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	var req model.SetTenantPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.SetTenantPolicy(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "tenant_policy_set_failed", "failed to set tenant policy")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *Handler) getOrListUserPolicies(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if tenantID != "" && userID != "" {
		item, err := h.svc.GetUserPolicyOverride(r.Context(), actor, tenantID, userID)
		if err != nil {
			status, code, message := service.ToResponse(err, "user_policy_get_failed", "failed to get user policy override")
			response.Error(w, r, status, code, message)
			return
		}
		response.JSON(w, http.StatusOK, item)
		return
	}

	q := model.ListUserPolicyOverridesQuery{
		ListQuery: model.ListQuery{
			Limit:  parseLimit(r.URL.Query().Get("limit")),
			Offset: parseOffset(r.URL.Query().Get("offset")),
		},
		TenantID: tenantID,
		UserID:   userID,
	}
	items, err := h.svc.ListUserPolicyOverrides(r.Context(), actor, q)
	if err != nil {
		status, code, message := service.ToResponse(err, "user_policy_list_failed", "failed to list user policy overrides")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) setUserPolicy(w http.ResponseWriter, r *http.Request) {
	actor := actorFromHeaders(r)
	var req model.SetUserPolicyOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	item, err := h.svc.SetUserPolicyOverride(r.Context(), actor, req)
	if err != nil {
		status, code, message := service.ToResponse(err, "user_policy_set_failed", "failed to set user policy override")
		response.Error(w, r, status, code, message)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func actorFromHeaders(r *http.Request) model.ActorPrincipal {
	rawScopes := strings.TrimSpace(r.Header.Get("X-Actor-Scopes"))
	scopes := make([]string, 0)
	if rawScopes != "" {
		for _, s := range strings.Split(rawScopes, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				scopes = append(scopes, s)
			}
		}
	}
	return model.ActorPrincipal{
		Subject:  strings.TrimSpace(r.Header.Get("X-Actor-Sub")),
		TenantID: strings.TrimSpace(r.Header.Get("X-Actor-Tenant-ID")),
		Scopes:   scopes,
	}
}

func parseLimit(raw string) int {
	if raw == "" {
		return 20
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 20
	}
	if n <= 0 || n > 100 {
		return 20
	}
	return n
}

func parseOffset(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseTimeQuery(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

