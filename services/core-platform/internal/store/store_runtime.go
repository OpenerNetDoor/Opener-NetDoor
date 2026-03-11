package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type runtimeStore interface {
	GetNodeRuntime(ctx context.Context, tenantID string, nodeID string) (model.NodeRuntime, error)
	UpsertNodeRuntimeDefaults(ctx context.Context, runtime model.NodeRuntime) (model.NodeRuntime, error)
	InsertRuntimeRevision(ctx context.Context, nodeID string, tenantID string, configJSON string, applied bool) (model.RuntimeRevision, error)
	MarkRuntimeRevisionApplied(ctx context.Context, nodeID string, version int) (model.RuntimeRevision, error)
	GetLatestRuntimeRevision(ctx context.Context, nodeID string, tenantID string) (model.RuntimeRevision, error)
}

func (s *SQLStore) GetNodeRuntime(ctx context.Context, tenantID string, nodeID string) (model.NodeRuntime, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT node_id::text, tenant_id::text, runtime_backend, runtime_protocol, listen_port,
		        reality_public_key, reality_short_id, reality_server_name,
		        applied_config_version, runtime_status, last_applied_at, COALESCE(last_error,''), created_at, updated_at
		 FROM node_runtimes
		 WHERE tenant_id = $1 AND node_id = $2`,
		tenantID,
		nodeID,
	)
	item, err := scanNodeRuntime(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeRuntime{}, sql.ErrNoRows
		}
		return model.NodeRuntime{}, fmt.Errorf("get node runtime: %w", err)
	}
	return item, nil
}

func (s *SQLStore) UpsertNodeRuntimeDefaults(ctx context.Context, runtime model.NodeRuntime) (model.NodeRuntime, error) {
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO node_runtimes (
		   node_id, tenant_id, runtime_backend, runtime_protocol, listen_port,
		   reality_public_key, reality_short_id, reality_server_name,
		   runtime_status, updated_at
		 ) VALUES (
		   $1, $2, $3, $4, $5,
		   $6, $7, $8,
		   'pending', NOW()
		 )
		 ON CONFLICT (node_id)
		 DO UPDATE SET
		   tenant_id = EXCLUDED.tenant_id,
		   runtime_backend = EXCLUDED.runtime_backend,
		   runtime_protocol = EXCLUDED.runtime_protocol,
		   listen_port = EXCLUDED.listen_port,
		   reality_public_key = EXCLUDED.reality_public_key,
		   reality_short_id = EXCLUDED.reality_short_id,
		   reality_server_name = EXCLUDED.reality_server_name,
		   updated_at = NOW()
		 RETURNING node_id::text, tenant_id::text, runtime_backend, runtime_protocol, listen_port,
		           reality_public_key, reality_short_id, reality_server_name,
		           applied_config_version, runtime_status, last_applied_at, COALESCE(last_error,''), created_at, updated_at`,
		runtime.NodeID,
		runtime.TenantID,
		runtime.RuntimeBackend,
		runtime.RuntimeProtocol,
		runtime.ListenPort,
		runtime.RealityPublicKey,
		runtime.RealityShortID,
		runtime.RealityServerName,
	)
	item, err := scanNodeRuntime(row.Scan)
	if err != nil {
		return model.NodeRuntime{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) InsertRuntimeRevision(ctx context.Context, nodeID string, tenantID string, configJSON string, applied bool) (model.RuntimeRevision, error) {
	row := s.db.QueryRowContext(
		ctx,
		`WITH next_version AS (
		   SELECT COALESCE(MAX(version), 0) + 1 AS version
		   FROM runtime_revisions
		   WHERE node_id = $1
		 )
		 INSERT INTO runtime_revisions (node_id, tenant_id, version, config_json, applied, applied_at)
		 SELECT $1, $2, next_version.version, $3::jsonb, $4,
		        CASE WHEN $4 THEN NOW() ELSE NULL END
		 FROM next_version
		 RETURNING id, node_id::text, tenant_id::text, version, config_json::text, applied, applied_at, created_at`,
		nodeID,
		tenantID,
		configJSON,
		applied,
	)
	item, err := scanRuntimeRevision(row.Scan)
	if err != nil {
		return model.RuntimeRevision{}, mapDBError(err)
	}
	if applied {
		_, err = s.db.ExecContext(
			ctx,
			`UPDATE node_runtimes
			 SET applied_config_version = $3,
			     runtime_status = 'active',
			     last_applied_at = NOW(),
			     last_error = NULL,
			     updated_at = NOW()
			 WHERE node_id = $1 AND tenant_id = $2`,
			nodeID,
			tenantID,
			item.Version,
		)
		if err != nil {
			return model.RuntimeRevision{}, mapDBError(err)
		}
	}
	return item, nil
}

func (s *SQLStore) MarkRuntimeRevisionApplied(ctx context.Context, nodeID string, version int) (model.RuntimeRevision, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.RuntimeRevision{}, fmt.Errorf("begin tx mark runtime revision applied: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE runtime_revisions
		 SET applied = FALSE, applied_at = NULL
		 WHERE node_id = $1`,
		nodeID,
	); err != nil {
		return model.RuntimeRevision{}, mapDBError(err)
	}

	row := tx.QueryRowContext(
		ctx,
		`UPDATE runtime_revisions
		 SET applied = TRUE, applied_at = NOW()
		 WHERE node_id = $1 AND version = $2
		 RETURNING id, node_id::text, tenant_id::text, version, config_json::text, applied, applied_at, created_at`,
		nodeID,
		version,
	)
	item, err := scanRuntimeRevision(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RuntimeRevision{}, sql.ErrNoRows
		}
		return model.RuntimeRevision{}, mapDBError(err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE node_runtimes
		 SET applied_config_version = $2,
		     runtime_status = 'active',
		     last_applied_at = NOW(),
		     last_error = NULL,
		     updated_at = NOW()
		 WHERE node_id = $1`,
		nodeID,
		version,
	); err != nil {
		return model.RuntimeRevision{}, mapDBError(err)
	}

	if err := tx.Commit(); err != nil {
		return model.RuntimeRevision{}, fmt.Errorf("commit mark runtime revision applied: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetLatestRuntimeRevision(ctx context.Context, nodeID string, tenantID string) (model.RuntimeRevision, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, node_id::text, tenant_id::text, version, config_json::text, applied, applied_at, created_at
		 FROM runtime_revisions
		 WHERE node_id = $1 AND tenant_id = $2
		 ORDER BY version DESC
		 LIMIT 1`,
		nodeID,
		tenantID,
	)
	item, err := scanRuntimeRevision(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RuntimeRevision{}, sql.ErrNoRows
		}
		return model.RuntimeRevision{}, fmt.Errorf("get latest runtime revision: %w", err)
	}
	return item, nil
}

func scanNodeRuntime(scan func(dest ...any) error) (model.NodeRuntime, error) {
	var out model.NodeRuntime
	var lastApplied sql.NullTime
	err := scan(
		&out.NodeID,
		&out.TenantID,
		&out.RuntimeBackend,
		&out.RuntimeProtocol,
		&out.ListenPort,
		&out.RealityPublicKey,
		&out.RealityShortID,
		&out.RealityServerName,
		&out.AppliedConfigVersion,
		&out.RuntimeStatus,
		&lastApplied,
		&out.LastError,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return model.NodeRuntime{}, err
	}
	if lastApplied.Valid {
		t := lastApplied.Time
		out.LastAppliedAt = &t
	}
	return out, nil
}

func scanRuntimeRevision(scan func(dest ...any) error) (model.RuntimeRevision, error) {
	var out model.RuntimeRevision
	var cfgRaw []byte
	var appliedAt sql.NullTime
	err := scan(&out.ID, &out.NodeID, &out.TenantID, &out.Version, &cfgRaw, &out.Applied, &appliedAt, &out.CreatedAt)
	if err != nil {
		return model.RuntimeRevision{}, err
	}
	if len(cfgRaw) > 0 {
		var compact map[string]any
		if unmarshalErr := json.Unmarshal(cfgRaw, &compact); unmarshalErr != nil {
			out.ConfigJSON = string(cfgRaw)
		} else {
			normalized, _ := json.Marshal(compact)
			out.ConfigJSON = string(normalized)
		}
	}
	if appliedAt.Valid {
		t := appliedAt.Time
		out.AppliedAt = &t
	}
	return out, nil
}
