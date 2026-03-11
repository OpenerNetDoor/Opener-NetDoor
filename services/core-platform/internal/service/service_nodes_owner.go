package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type nodeOwnerStore interface {
	CreateNode(ctx context.Context, in model.CreateNodeRequest) (model.Node, error)
}

func (s *CoreService) CreateNode(ctx context.Context, actor model.ActorPrincipal, in model.CreateNodeRequest) (model.Node, error) {
	in.TenantID = strings.TrimSpace(in.TenantID)
	if in.TenantID == "" && actor.TenantID != "" {
		in.TenantID = actor.TenantID
	}
	in.Region = strings.TrimSpace(in.Region)
	in.Hostname = strings.TrimSpace(in.Hostname)
	in.AgentVersion = strings.TrimSpace(in.AgentVersion)
	in.ContractVersion = strings.TrimSpace(in.ContractVersion)

	if in.TenantID == "" || in.Region == "" || in.Hostname == "" {
		return model.Node{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id, region and hostname are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.Node{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if in.AgentVersion == "" {
		in.AgentVersion = "owner-manual"
	}
	if in.ContractVersion == "" {
		in.ContractVersion = s.opts.NodeContractVersion
	}
	if len(in.Capabilities) == 0 {
		in.Capabilities = append([]string(nil), s.opts.NodeRequiredCapabilities...)
	}

	for i := range in.Capabilities {
		in.Capabilities[i] = strings.TrimSpace(in.Capabilities[i])
	}
	in.Capabilities = compactStrings(in.Capabilities)
	if len(in.Capabilities) == 0 {
		return model.Node{}, &AppError{Status: 400, Code: "validation_error", Message: "capabilities cannot be empty"}
	}

	creator, ok := s.store.(nodeOwnerStore)
	if !ok {
		return model.Node{}, &AppError{Status: 500, Code: "node_create_not_supported", Message: "node create is not supported by configured store"}
	}
	item, err := creator.CreateNode(ctx, in)
	if err != nil {
		return model.Node{}, mapStoreError("node_create_failed", err)
	}
	item.Status = s.deriveNodeStatus(item, time.Now().UTC())

	runtimeItem, runtimeErr := s.ensureNodeRuntimeDefaults(ctx, item)
	if runtimeErr != nil {
		return model.Node{}, runtimeErr
	}

	if err := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
		TenantID:   item.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "node.created",
		TargetType: "node",
		TargetID:   item.ID,
		Metadata: map[string]any{
			"hostname":         item.Hostname,
			"region":           item.Region,
			"node_key_id":      item.NodeKeyID,
			"source":           "owner_panel",
			"runtime_backend":  runtimeItem.RuntimeBackend,
			"runtime_protocol": runtimeItem.RuntimeProtocol,
		},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return model.Node{}, mapStoreError("node_create_failed", err)
	}

	return item, nil
}

func (s *CoreService) GetNode(ctx context.Context, actor model.ActorPrincipal, tenantID string, nodeID string) (model.Node, error) {
	tenantID = strings.TrimSpace(tenantID)
	nodeID = strings.TrimSpace(nodeID)
	if tenantID == "" && actor.TenantID != "" {
		tenantID = actor.TenantID
	}
	if tenantID == "" || nodeID == "" {
		return model.Node{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(tenantID) {
		return model.Node{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	item, err := s.store.GetNodeByID(ctx, tenantID, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.Node{}, &AppError{Status: 500, Code: "node_get_failed", Message: "failed to get node", Err: err}
	}
	item.Status = s.deriveNodeStatus(item, time.Now().UTC())
	return item, nil
}

func compactStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
