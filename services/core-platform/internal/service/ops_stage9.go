package service

import (
	"context"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func (s *CoreService) ListAuditLogs(ctx context.Context, actor model.ActorPrincipal, q model.ListAuditLogsQuery) ([]model.AuditLogRecord, error) {
	q.TenantID = strings.TrimSpace(q.TenantID)
	if !actor.IsPlatformAdmin() {
		if q.TenantID == "" {
			q.TenantID = actor.TenantID
		}
	}
	if q.TenantID != "" && !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	if q.Since != nil && q.Until != nil && q.Since.After(*q.Until) {
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "since must be before until"}
	}

	items, err := s.store.ListAuditLogs(ctx, q)
	if err != nil {
		return nil, &AppError{Status: 500, Code: "audit_log_list_failed", Message: "failed to list audit logs", Err: err}
	}
	return items, nil
}

func (s *CoreService) GetOpsSnapshot(ctx context.Context, actor model.ActorPrincipal, tenantID string) (model.OpsSnapshot, error) {
	tenantID = strings.TrimSpace(tenantID)
	if !actor.IsPlatformAdmin() {
		if tenantID == "" {
			tenantID = actor.TenantID
		}
		if !actor.CanAccessTenant(tenantID) {
			return model.OpsSnapshot{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
		}
	}
	if tenantID != "" && !actor.CanAccessTenant(tenantID) {
		return model.OpsSnapshot{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	now := time.Now().UTC()
	since := now.Add(-24 * time.Hour)
	expiringBefore := now.Add(24 * time.Hour)

	nodeStatus, err := s.store.ListNodeStatusCounts(ctx, tenantID)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load node status counters", Err: err}
	}
	activeCertificates, err := s.store.CountActiveNodeCertificates(ctx, tenantID)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load active certificate counters", Err: err}
	}
	expiringCertificates, err := s.store.CountExpiringNodeCertificates(ctx, tenantID, expiringBefore)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load expiring certificate counters", Err: err}
	}
	trafficBytes24h, err := s.store.GetTrafficUsageTotalBetween(ctx, tenantID, since, now)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load traffic usage counters", Err: err}
	}
	replayRejected24h, err := s.store.CountAuditActionsSince(ctx, tenantID, "node.replay_rejected", since)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load replay counters", Err: err}
	}
	invalidSignature24h, err := s.store.CountAuditActionsSince(ctx, tenantID, "node.invalid_signature", since)
	if err != nil {
		return model.OpsSnapshot{}, &AppError{Status: 500, Code: "ops_snapshot_failed", Message: "failed to load invalid signature counters", Err: err}
	}

	return model.OpsSnapshot{
		TenantID:             tenantID,
		GeneratedAt:          now,
		NodeStatus:           nodeStatus,
		ActiveCertificates:   activeCertificates,
		ExpiringCertificates: expiringCertificates,
		TrafficBytes24h:      trafficBytes24h,
		ReplayRejected24h:    replayRejected24h,
		InvalidSignature24h:  invalidSignature24h,
	}, nil
}
