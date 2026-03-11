package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type runtimeStore interface {
	GetNodeRuntime(ctx context.Context, tenantID string, nodeID string) (model.NodeRuntime, error)
	UpsertNodeRuntimeDefaults(ctx context.Context, runtime model.NodeRuntime) (model.NodeRuntime, error)
	InsertRuntimeRevision(ctx context.Context, nodeID string, tenantID string, configJSON string, applied bool) (model.RuntimeRevision, error)
	MarkRuntimeRevisionApplied(ctx context.Context, nodeID string, version int) (model.RuntimeRevision, error)
	GetLatestRuntimeRevision(ctx context.Context, nodeID string, tenantID string) (model.RuntimeRevision, error)
}

type runtimeOwnerService interface {
	GetNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.RuntimeConfigResponse, error)
	ApplyNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, in model.RuntimeApplyRequest) (model.RuntimeConfigResponse, error)
}

func (s *CoreService) ensureNodeRuntimeDefaults(ctx context.Context, node model.Node) (model.NodeRuntime, error) {
	rs, ok := s.store.(runtimeStore)
	if !ok {
		return model.NodeRuntime{}, &AppError{Status: 500, Code: "runtime_store_not_supported", Message: "runtime store is not supported"}
	}

	runtime := model.NodeRuntime{
		NodeID:            node.ID,
		TenantID:          node.TenantID,
		RuntimeBackend:    "xray",
		RuntimeProtocol:   "vless_reality",
		ListenPort:        s.opts.RuntimeVLESSPort,
		RealityPublicKey:  s.opts.RuntimeRealityPublicKey,
		RealityShortID:    s.opts.RuntimeRealityShortID,
		RealityServerName: s.opts.RuntimeRealityServerName,
	}
	if strings.TrimSpace(runtime.RealityPublicKey) == "" {
		// Runtime not configured yet; keep placeholder runtime row for visibility.
		runtime.RealityPublicKey = "pending"
	}
	item, err := rs.UpsertNodeRuntimeDefaults(ctx, runtime)
	if err != nil {
		return model.NodeRuntime{}, mapStoreError("runtime_defaults_failed", err)
	}
	return item, nil
}

func (s *CoreService) enrichAccessKey(ctx context.Context, item model.AccessKey) model.AccessKey {
	if strings.ToLower(strings.TrimSpace(item.KeyType)) != "vless" {
		return item
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(item.SecretRef)), "vless://") {
		item.ConnectionURI = item.SecretRef
		return item
	}

	node, runtime, ok := s.resolveRuntimeTarget(ctx, item.TenantID)
	if !ok {
		return item
	}

	uuid := strings.TrimSpace(item.SecretRef)
	if !uuidRe.MatchString(uuid) {
		return item
	}
	item.ConnectionURI = buildVLESSRealityURI(node, runtime, uuid, item.ID)
	return item
}

func (s *CoreService) resolveRuntimeTarget(ctx context.Context, tenantID string) (model.Node, model.NodeRuntime, bool) {
	nodes, err := s.store.ListNodes(ctx, model.ListNodesQuery{
		ListQuery: model.ListQuery{Limit: 100, Offset: 0},
		TenantID:  tenantID,
	})
	if err != nil || len(nodes) == 0 {
		return model.Node{}, model.NodeRuntime{}, false
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return statusPriority(nodes[i].Status) < statusPriority(nodes[j].Status)
	})

	rs, ok := s.store.(runtimeStore)
	if !ok {
		return model.Node{}, model.NodeRuntime{}, false
	}
	for _, n := range nodes {
		runtime, runtimeErr := rs.GetNodeRuntime(ctx, tenantID, n.ID)
		if runtimeErr != nil {
			continue
		}
		if strings.TrimSpace(runtime.RealityPublicKey) == "" || runtime.RealityPublicKey == "pending" {
			continue
		}
		return n, runtime, true
	}
	return model.Node{}, model.NodeRuntime{}, false
}

func statusPriority(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 0
	case "pending":
		return 1
	case "stale":
		return 2
	default:
		return 10
	}
}

func buildVLESSRealityURI(node model.Node, runtime model.NodeRuntime, uuid string, keyID string) string {
	host := strings.TrimSpace(node.Hostname)
	if host == "" {
		host = "127.0.0.1"
	}
	query := url.Values{}
	query.Set("encryption", "none")
	query.Set("security", "reality")
	query.Set("type", "tcp")
	query.Set("sni", runtime.RealityServerName)
	query.Set("fp", "chrome")
	query.Set("pbk", runtime.RealityPublicKey)
	query.Set("sid", runtime.RealityShortID)
	query.Set("flow", "xtls-rprx-vision")
	suffix := keyID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	fragment := url.QueryEscape(fmt.Sprintf("%s-%s", node.Region, suffix))
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s", uuid, host, runtime.ListenPort, query.Encode(), fragment)
}

func maybeGenerateAccessSecretRef(keyType string, current string) (string, error) {
	if strings.TrimSpace(current) != "" {
		return current, nil
	}
	if strings.ToLower(strings.TrimSpace(keyType)) != "vless" {
		return "secret://generated/" + strconv.FormatInt(time.Now().UnixNano(), 10), nil
	}
	return randomUUIDv4()
}

func randomUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	hexID := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexID[0:8], hexID[8:12], hexID[12:16], hexID[16:20], hexID[20:32]), nil
}

func (s *CoreService) GetNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.RuntimeConfigResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	nodeID = strings.TrimSpace(nodeID)
	if tenantID == "" || nodeID == "" {
		return model.RuntimeConfigResponse{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(tenantID) {
		return model.RuntimeConfigResponse{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	node, err := s.store.GetNodeByID(ctx, tenantID, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RuntimeConfigResponse{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_config_failed", err)
	}
	runtime, err := s.ensureNodeRuntimeDefaults(ctx, node)
	if err != nil {
		return model.RuntimeConfigResponse{}, err
	}

	configJSON, err := s.generateXrayConfig(ctx, node, runtime)
	if err != nil {
		return model.RuntimeConfigResponse{}, err
	}

	rs, ok := s.store.(runtimeStore)
	if !ok {
		return model.RuntimeConfigResponse{}, &AppError{Status: 500, Code: "runtime_store_not_supported", Message: "runtime store is not supported"}
	}
	rev, err := rs.InsertRuntimeRevision(ctx, node.ID, node.TenantID, configJSON, false)
	if err != nil {
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_config_failed", err)
	}
	return model.RuntimeConfigResponse{Node: node, Runtime: runtime, Revision: rev, ConfigJSON: configJSON}, nil
}

func (s *CoreService) ApplyNodeRuntimeConfig(ctx context.Context, actor model.ActorPrincipal, in model.RuntimeApplyRequest) (model.RuntimeConfigResponse, error) {
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.NodeID = strings.TrimSpace(in.NodeID)
	if in.TenantID == "" || in.NodeID == "" {
		return model.RuntimeConfigResponse{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.RuntimeConfigResponse{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	rs, ok := s.store.(runtimeStore)
	if !ok {
		return model.RuntimeConfigResponse{}, &AppError{Status: 500, Code: "runtime_store_not_supported", Message: "runtime store is not supported"}
	}

	latest, err := rs.GetLatestRuntimeRevision(ctx, in.NodeID, in.TenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RuntimeConfigResponse{}, &AppError{Status: 404, Code: "runtime_revision_not_found", Message: "runtime revision not found", Err: err}
		}
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_apply_failed", err)
	}
	applied, err := rs.MarkRuntimeRevisionApplied(ctx, in.NodeID, latest.Version)
	if err != nil {
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_apply_failed", err)
	}

	node, err := s.store.GetNodeByID(ctx, in.TenantID, in.NodeID)
	if err != nil {
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_apply_failed", err)
	}
	runtime, err := rs.GetNodeRuntime(ctx, in.TenantID, in.NodeID)
	if err != nil {
		return model.RuntimeConfigResponse{}, mapStoreError("runtime_apply_failed", err)
	}

	s.auditBestEffort(ctx, model.AuditLogEvent{
		TenantID:   in.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "runtime.applied",
		TargetType: "node",
		TargetID:   in.NodeID,
		Metadata: map[string]any{
			"version": applied.Version,
		},
	})

	return model.RuntimeConfigResponse{Node: node, Runtime: runtime, Revision: applied, ConfigJSON: applied.ConfigJSON}, nil
}

func (s *CoreService) generateXrayConfig(ctx context.Context, node model.Node, runtime model.NodeRuntime) (string, error) {
	keys, err := s.store.ListAccessKeys(ctx, model.ListAccessKeysQuery{
		ListQuery: model.ListQuery{Limit: 10000, Offset: 0, Status: "active"},
		TenantID:  node.TenantID,
	})
	if err != nil {
		return "", mapStoreError("runtime_config_failed", err)
	}
	clients := make([]map[string]any, 0)
	for _, key := range keys {
		if strings.ToLower(strings.TrimSpace(key.KeyType)) != "vless" {
			continue
		}
		if !uuidRe.MatchString(strings.TrimSpace(key.SecretRef)) {
			continue
		}
		clients = append(clients, map[string]any{
			"id":    key.SecretRef,
			"email": fmt.Sprintf("%s:%s", key.UserID, key.ID),
			"flow":  "xtls-rprx-vision",
		})
	}
	config := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds": []map[string]any{
			{
				"tag":      "vless-reality-in",
				"port":     runtime.ListenPort,
				"protocol": "vless",
				"settings": map[string]any{
					"clients":    clients,
					"decryption": "none",
				},
				"streamSettings": map[string]any{
					"network":  "tcp",
					"security": "reality",
					"realitySettings": map[string]any{
						"show":        false,
						"dest":        runtime.RealityServerName + ":443",
						"serverNames": []string{runtime.RealityServerName},
						"privateKey":  s.opts.RuntimeRealityPrivateKey,
						"shortIds":    []string{runtime.RealityShortID},
					},
				},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []string{"http", "tls", "quic"},
				},
			},
		},
		"outbounds": []map[string]any{
			{"tag": "direct", "protocol": "freedom"},
			{"tag": "block", "protocol": "blackhole"},
		},
		"routing": map[string]any{
			"rules": []map[string]any{},
		},
	}
	blob, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", &AppError{Status: 500, Code: "runtime_config_failed", Message: "failed to encode xray config", Err: err}
	}
	return string(blob), nil
}
