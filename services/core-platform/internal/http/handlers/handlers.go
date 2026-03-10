package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/http/response"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/service"
)

type Handler struct {
	svc service.Service
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
	default:
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
	}
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
	if r.Method != http.MethodGet {
		response.Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "unsupported method")
		return
	}
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
