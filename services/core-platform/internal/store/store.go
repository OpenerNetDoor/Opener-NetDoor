package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type ErrorKind string

const (
	ErrorKindConflict   ErrorKind = "conflict"
	ErrorKindForeignKey ErrorKind = "foreign_key"
	ErrorKindValidation ErrorKind = "validation"
)

type DBError struct {
	Kind       ErrorKind
	Constraint string
	Message    string
	Err        error
}

func (e *DBError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "database error"
}

func (e *DBError) Unwrap() error { return e.Err }

type Store interface {
	ListTenants(ctx context.Context, q model.ListQuery) ([]model.Tenant, error)
	GetTenantByID(ctx context.Context, tenantID string) (model.Tenant, error)
	CreateTenant(ctx context.Context, in model.CreateTenantRequest) (model.Tenant, error)

	ListUsers(ctx context.Context, q model.ListUsersQuery) ([]model.User, error)
	CreateUser(ctx context.Context, in model.CreateUserRequest) (model.User, error)

	ListAccessKeys(ctx context.Context, q model.ListAccessKeysQuery) ([]model.AccessKey, error)
	CreateAccessKey(ctx context.Context, in model.CreateAccessKeyRequest) (model.AccessKey, error)
	RevokeAccessKey(ctx context.Context, keyID string, tenantID string) (model.AccessKey, error)

	ListTenantPolicies(ctx context.Context, q model.ListTenantPoliciesQuery) ([]model.TenantPolicy, error)
	GetTenantPolicy(ctx context.Context, tenantID string) (model.TenantPolicy, error)
	UpsertTenantPolicy(ctx context.Context, actor model.ActorPrincipal, in model.SetTenantPolicyRequest) (model.TenantPolicy, error)

	ListUserPolicyOverrides(ctx context.Context, q model.ListUserPolicyOverridesQuery) ([]model.UserPolicyOverride, error)
	GetUserPolicyOverride(ctx context.Context, tenantID string, userID string) (model.UserPolicyOverride, error)
	UpsertUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, in model.SetUserPolicyOverrideRequest) (model.UserPolicyOverride, error)

	GetEffectivePolicy(ctx context.Context, tenantID string, userID string) (model.EffectivePolicy, error)
	GetTenantUsageTotal(ctx context.Context, tenantID string) (int64, error)

	GetDeviceByFingerprint(ctx context.Context, tenantID string, fingerprint string) (model.Device, error)
	CountActiveDevicesForUser(ctx context.Context, tenantID string, userID string) (int, error)
	RegisterDevice(ctx context.Context, in model.RegisterDeviceRequest) (model.Device, error)

	ListNodes(ctx context.Context, q model.ListNodesQuery) ([]model.Node, error)
	GetNodeByID(ctx context.Context, tenantID string, nodeID string) (model.Node, error)
	GetNodeByKey(ctx context.Context, tenantID string, nodeKeyID string) (model.Node, error)
	UpsertNodeRegistration(ctx context.Context, in model.RegisterNodeRequest, identityFingerprint string) (model.Node, error)
	TouchNodeHeartbeat(ctx context.Context, in model.NodeHeartbeatRequest) (model.Node, error)
	InsertNodeHeartbeatEvent(ctx context.Context, nodeID string, tenantID string, status string, metadata map[string]any) error

	Ping(ctx context.Context) error
	Close() error
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(databaseURL string) (*SQLStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

func (s *SQLStore) ListTenants(ctx context.Context, q model.ListQuery) ([]model.Tenant, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	status := strings.TrimSpace(q.Status)

	base := `SELECT id::text, name, status, created_at FROM tenants`
	args := make([]any, 0, 3)
	if status != "" {
		base += ` WHERE status = $1`
		args = append(args, status)
	}
	base += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	out := make([]model.Tenant, 0)
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenants: %w", err)
	}
	return out, nil
}

func (s *SQLStore) GetTenantByID(ctx context.Context, tenantID string) (model.Tenant, error) {
	var t model.Tenant
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, name, status, created_at FROM tenants WHERE id = $1`,
		tenantID,
	).Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Tenant{}, sql.ErrNoRows
		}
		return model.Tenant{}, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}

func (s *SQLStore) CreateTenant(ctx context.Context, in model.CreateTenantRequest) (model.Tenant, error) {
	var t model.Tenant
	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO tenants (name) VALUES ($1) RETURNING id::text, name, status, created_at`,
		in.Name,
	).Scan(&t.ID, &t.Name, &t.Status, &t.CreatedAt)
	if err != nil {
		return model.Tenant{}, mapDBError(err)
	}
	return t, nil
}

func (s *SQLStore) ListUsers(ctx context.Context, q model.ListUsersQuery) ([]model.User, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id::text, tenant_id::text, COALESCE(email,''), status, COALESCE(note,''), created_at
		 FROM users
		 WHERE tenant_id = $1
		   AND ($2 = '' OR status = $2)
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		q.TenantID,
		strings.TrimSpace(q.Status),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	out := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Status, &u.Note, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

func (s *SQLStore) CreateUser(ctx context.Context, in model.CreateUserRequest) (model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO users (tenant_id, email, note) VALUES ($1, $2, $3)
		 RETURNING id::text, tenant_id::text, COALESCE(email,''), status, COALESCE(note,''), created_at`,
		in.TenantID, in.Email, in.Note,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Status, &u.Note, &u.CreatedAt)
	if err != nil {
		return model.User{}, mapDBError(err)
	}
	return u, nil
}

func (s *SQLStore) ListAccessKeys(ctx context.Context, q model.ListAccessKeysQuery) ([]model.AccessKey, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id::text, tenant_id::text, user_id::text, key_type, secret_ref, status, expires_at, created_at
		 FROM access_keys
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ($2 = '' OR user_id::text = $2)
		   AND ($3 = '' OR status = $3)
		 ORDER BY created_at DESC
		 LIMIT $4 OFFSET $5`,
		strings.TrimSpace(q.TenantID),
		strings.TrimSpace(q.UserID),
		strings.TrimSpace(q.Status),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query access keys: %w", err)
	}
	defer rows.Close()

	out := make([]model.AccessKey, 0)
	for rows.Next() {
		var k model.AccessKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.UserID, &k.KeyType, &k.SecretRef, &k.Status, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan access key: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access keys: %w", err)
	}
	return out, nil
}

func (s *SQLStore) CreateAccessKey(ctx context.Context, in model.CreateAccessKeyRequest) (model.AccessKey, error) {
	var k model.AccessKey
	err := s.db.QueryRowContext(
		ctx,
		`INSERT INTO access_keys (tenant_id, user_id, key_type, secret_ref, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id::text, tenant_id::text, user_id::text, key_type, secret_ref, status, expires_at, created_at`,
		in.TenantID, in.UserID, in.KeyType, in.SecretRef, in.ExpiresAt,
	).Scan(&k.ID, &k.TenantID, &k.UserID, &k.KeyType, &k.SecretRef, &k.Status, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return model.AccessKey{}, mapDBError(err)
	}
	return k, nil
}

func (s *SQLStore) RevokeAccessKey(ctx context.Context, keyID string, tenantID string) (model.AccessKey, error) {
	var k model.AccessKey
	query := `UPDATE access_keys SET status = 'revoked' WHERE id = $1`
	args := []any{keyID}
	if strings.TrimSpace(tenantID) != "" {
		query += ` AND tenant_id::text = $2`
		args = append(args, tenantID)
	}
	query += ` RETURNING id::text, tenant_id::text, user_id::text, key_type, secret_ref, status, expires_at, created_at`

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&k.ID, &k.TenantID, &k.UserID, &k.KeyType, &k.SecretRef, &k.Status, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AccessKey{}, sql.ErrNoRows
		}
		return model.AccessKey{}, mapDBError(err)
	}
	return k, nil
}

func (s *SQLStore) ListTenantPolicies(ctx context.Context, q model.ListTenantPoliciesQuery) ([]model.TenantPolicy, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT tenant_id::text, traffic_quota_bytes, device_limit, default_key_ttl_seconds, COALESCE(updated_by,''), updated_at
		 FROM tenant_policies
		 WHERE ($1 = '' OR tenant_id::text = $1)
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		strings.TrimSpace(q.TenantID),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query tenant policies: %w", err)
	}
	defer rows.Close()

	out := make([]model.TenantPolicy, 0)
	for rows.Next() {
		item, err := scanTenantPolicy(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan tenant policy: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant policies: %w", err)
	}
	return out, nil
}

func (s *SQLStore) GetTenantPolicy(ctx context.Context, tenantID string) (model.TenantPolicy, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT tenant_id::text, traffic_quota_bytes, device_limit, default_key_ttl_seconds, COALESCE(updated_by,''), updated_at
		 FROM tenant_policies
		 WHERE tenant_id = $1`,
		tenantID,
	)
	item, err := scanTenantPolicy(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.TenantPolicy{}, sql.ErrNoRows
		}
		return model.TenantPolicy{}, fmt.Errorf("get tenant policy: %w", err)
	}
	return item, nil
}

func (s *SQLStore) UpsertTenantPolicy(ctx context.Context, actor model.ActorPrincipal, in model.SetTenantPolicyRequest) (model.TenantPolicy, error) {
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO tenant_policies (tenant_id, traffic_quota_bytes, device_limit, default_key_ttl_seconds, updated_by, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (tenant_id)
		 DO UPDATE SET
		   traffic_quota_bytes = EXCLUDED.traffic_quota_bytes,
		   device_limit = EXCLUDED.device_limit,
		   default_key_ttl_seconds = EXCLUDED.default_key_ttl_seconds,
		   updated_by = EXCLUDED.updated_by,
		   updated_at = NOW()
		 RETURNING tenant_id::text, traffic_quota_bytes, device_limit, default_key_ttl_seconds, COALESCE(updated_by,''), updated_at`,
		in.TenantID,
		in.TrafficQuotaBytes,
		in.DeviceLimit,
		in.DefaultKeyTTLSeconds,
		actor.Subject,
	)
	item, err := scanTenantPolicy(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.TenantPolicy{}, sql.ErrNoRows
		}
		return model.TenantPolicy{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) ListUserPolicyOverrides(ctx context.Context, q model.ListUserPolicyOverridesQuery) ([]model.UserPolicyOverride, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT tenant_id::text, user_id::text, traffic_quota_bytes, device_limit, key_ttl_seconds, COALESCE(updated_by,''), updated_at
		 FROM user_policy_overrides
		 WHERE tenant_id::text = $1
		   AND ($2 = '' OR user_id::text = $2)
		 ORDER BY updated_at DESC
		 LIMIT $3 OFFSET $4`,
		strings.TrimSpace(q.TenantID),
		strings.TrimSpace(q.UserID),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query user policy overrides: %w", err)
	}
	defer rows.Close()

	out := make([]model.UserPolicyOverride, 0)
	for rows.Next() {
		item, err := scanUserPolicyOverride(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan user policy override: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user policy overrides: %w", err)
	}
	return out, nil
}

func (s *SQLStore) GetUserPolicyOverride(ctx context.Context, tenantID string, userID string) (model.UserPolicyOverride, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT tenant_id::text, user_id::text, traffic_quota_bytes, device_limit, key_ttl_seconds, COALESCE(updated_by,''), updated_at
		 FROM user_policy_overrides
		 WHERE tenant_id = $1 AND user_id = $2`,
		tenantID,
		userID,
	)
	item, err := scanUserPolicyOverride(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.UserPolicyOverride{}, sql.ErrNoRows
		}
		return model.UserPolicyOverride{}, fmt.Errorf("get user policy override: %w", err)
	}
	return item, nil
}

func (s *SQLStore) UpsertUserPolicyOverride(ctx context.Context, actor model.ActorPrincipal, in model.SetUserPolicyOverrideRequest) (model.UserPolicyOverride, error) {
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO user_policy_overrides (tenant_id, user_id, traffic_quota_bytes, device_limit, key_ttl_seconds, updated_by, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (tenant_id, user_id)
		 DO UPDATE SET
		   traffic_quota_bytes = EXCLUDED.traffic_quota_bytes,
		   device_limit = EXCLUDED.device_limit,
		   key_ttl_seconds = EXCLUDED.key_ttl_seconds,
		   updated_by = EXCLUDED.updated_by,
		   updated_at = NOW()
		 RETURNING tenant_id::text, user_id::text, traffic_quota_bytes, device_limit, key_ttl_seconds, COALESCE(updated_by,''), updated_at`,
		in.TenantID,
		in.UserID,
		in.TrafficQuotaBytes,
		in.DeviceLimit,
		in.KeyTTLSeconds,
		actor.Subject,
	)
	item, err := scanUserPolicyOverride(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.UserPolicyOverride{}, sql.ErrNoRows
		}
		return model.UserPolicyOverride{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) GetEffectivePolicy(ctx context.Context, tenantID string, userID string) (model.EffectivePolicy, error) {
	var out model.EffectivePolicy
	var trafficQuota sql.NullInt64
	var deviceLimit sql.NullInt64
	var keyTTL sql.NullInt64

	err := s.db.QueryRowContext(
		ctx,
		`SELECT
		  u.tenant_id::text,
		  u.id::text,
		  COALESCE(upo.traffic_quota_bytes, tp.traffic_quota_bytes) AS traffic_quota_bytes,
		  COALESCE(upo.device_limit, tp.device_limit) AS device_limit,
		  COALESCE(upo.key_ttl_seconds, tp.default_key_ttl_seconds) AS key_ttl_seconds,
		  CASE WHEN upo.user_id IS NULL THEN 'tenant_default' ELSE 'user_override' END AS source
		 FROM users u
		 LEFT JOIN tenant_policies tp ON tp.tenant_id = u.tenant_id
		 LEFT JOIN user_policy_overrides upo ON upo.tenant_id = u.tenant_id AND upo.user_id = u.id
		 WHERE u.tenant_id = $1 AND u.id = $2`,
		tenantID,
		userID,
	).Scan(&out.TenantID, &out.UserID, &trafficQuota, &deviceLimit, &keyTTL, &out.Source)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.EffectivePolicy{}, sql.ErrNoRows
		}
		return model.EffectivePolicy{}, fmt.Errorf("get effective policy: %w", err)
	}
	out.TrafficQuotaBytes = int64PtrFromNull(trafficQuota)
	out.DeviceLimit = intPtrFromNull(deviceLimit)
	out.KeyTTLSeconds = intPtrFromNull(keyTTL)
	return out, nil
}

func (s *SQLStore) GetTenantUsageTotal(ctx context.Context, tenantID string) (int64, error) {
	var total sql.NullInt64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(bytes_in + bytes_out), 0) FROM traffic_usage_hourly WHERE tenant_id = $1`,
		tenantID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get tenant usage total: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func (s *SQLStore) GetDeviceByFingerprint(ctx context.Context, tenantID string, fingerprint string) (model.Device, error) {
	var d model.Device
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, user_id::text, device_fingerprint, platform, status, created_at
		 FROM devices
		 WHERE tenant_id = $1 AND device_fingerprint = $2`,
		tenantID,
		fingerprint,
	).Scan(&d.ID, &d.TenantID, &d.UserID, &d.DeviceFingerprint, &d.Platform, &d.Status, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Device{}, sql.ErrNoRows
		}
		return model.Device{}, fmt.Errorf("get device by fingerprint: %w", err)
	}
	return d, nil
}

func (s *SQLStore) CountActiveDevicesForUser(ctx context.Context, tenantID string, userID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM devices
		 WHERE tenant_id = $1 AND user_id = $2 AND status = 'active'`,
		tenantID,
		userID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active devices: %w", err)
	}
	return n, nil
}

func (s *SQLStore) RegisterDevice(ctx context.Context, in model.RegisterDeviceRequest) (model.Device, error) {
	existing, err := s.GetDeviceByFingerprint(ctx, in.TenantID, in.DeviceFingerprint)
	if err == nil {
		if existing.UserID != in.UserID {
			return model.Device{}, &DBError{Kind: ErrorKindConflict, Message: "device fingerprint already bound to another user"}
		}

		var out model.Device
		err = s.db.QueryRowContext(
			ctx,
			`UPDATE devices
			 SET platform = $3, status = 'active'
			 WHERE tenant_id = $1 AND user_id = $2 AND device_fingerprint = $4
			 RETURNING id::text, tenant_id::text, user_id::text, device_fingerprint, platform, status, created_at`,
			in.TenantID,
			in.UserID,
			in.Platform,
			in.DeviceFingerprint,
		).Scan(&out.ID, &out.TenantID, &out.UserID, &out.DeviceFingerprint, &out.Platform, &out.Status, &out.CreatedAt)
		if err != nil {
			return model.Device{}, mapDBError(err)
		}
		return out, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.Device{}, err
	}

	var out model.Device
	err = s.db.QueryRowContext(
		ctx,
		`INSERT INTO devices (tenant_id, user_id, device_fingerprint, platform)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, tenant_id::text, user_id::text, device_fingerprint, platform, status, created_at`,
		in.TenantID,
		in.UserID,
		in.DeviceFingerprint,
		in.Platform,
	).Scan(&out.ID, &out.TenantID, &out.UserID, &out.DeviceFingerprint, &out.Platform, &out.Status, &out.CreatedAt)
	if err != nil {
		return model.Device{}, mapDBError(err)
	}
	return out, nil
}

func (s *SQLStore) ListNodes(ctx context.Context, q model.ListNodesQuery) ([]model.Node, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		        capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at
		 FROM nodes
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ($2 = '' OR status = $2)
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		strings.TrimSpace(q.TenantID),
		strings.TrimSpace(q.Status),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	out := make([]model.Node, 0)
	for rows.Next() {
		item, scanErr := scanNode(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan node: %w", scanErr)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}
	return out, nil
}

func (s *SQLStore) GetNodeByID(ctx context.Context, tenantID string, nodeID string) (model.Node, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		        capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at
		 FROM nodes
		 WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		nodeID,
	)
	item, err := scanNode(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, sql.ErrNoRows
		}
		return model.Node{}, fmt.Errorf("get node by id: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetNodeByKey(ctx context.Context, tenantID string, nodeKeyID string) (model.Node, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		        capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at
		 FROM nodes
		 WHERE tenant_id = $1 AND node_key_id = $2`,
		tenantID,
		nodeKeyID,
	)
	item, err := scanNode(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, sql.ErrNoRows
		}
		return model.Node{}, fmt.Errorf("get node by key: %w", err)
	}
	return item, nil
}

func (s *SQLStore) UpsertNodeRegistration(ctx context.Context, in model.RegisterNodeRequest, identityFingerprint string) (model.Node, error) {
	capsJSON, err := json.Marshal(in.Capabilities)
	if err != nil {
		return model.Node{}, fmt.Errorf("marshal node capabilities: %w", err)
	}

	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO nodes (
		   tenant_id, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		   capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at
		 ) VALUES (
		   $1, $2, $3, $4, $5, $6, $7,
		   $8::jsonb, $9, 'pending', NOW(), NULL
		 )
		 ON CONFLICT (tenant_id, node_key_id)
		 DO UPDATE SET
		   region = EXCLUDED.region,
		   hostname = EXCLUDED.hostname,
		   node_public_key = EXCLUDED.node_public_key,
		   contract_version = EXCLUDED.contract_version,
		   agent_version = EXCLUDED.agent_version,
		   capabilities = EXCLUDED.capabilities,
		   identity_fingerprint = EXCLUDED.identity_fingerprint,
		   last_seen_at = NOW(),
		   status = CASE WHEN nodes.status = 'revoked' THEN nodes.status ELSE 'pending' END
		 RETURNING id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		           capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at`,
		in.TenantID,
		in.Region,
		in.Hostname,
		in.NodeKeyID,
		in.NodePublicKey,
		in.ContractVersion,
		in.AgentVersion,
		string(capsJSON),
		identityFingerprint,
	)
	item, scanErr := scanNode(row.Scan)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return model.Node{}, sql.ErrNoRows
		}
		return model.Node{}, mapDBError(scanErr)
	}
	return item, nil
}

func (s *SQLStore) TouchNodeHeartbeat(ctx context.Context, in model.NodeHeartbeatRequest) (model.Node, error) {
	row := s.db.QueryRowContext(
		ctx,
		`UPDATE nodes
		 SET last_heartbeat_at = NOW(),
		     last_seen_at = NOW(),
		     status = 'active',
		     contract_version = $4,
		     agent_version = $5
		 WHERE tenant_id = $1
		   AND id = $2
		   AND node_key_id = $3
		   AND status <> 'revoked'
		 RETURNING id::text, tenant_id::text, region, hostname, node_key_id, node_public_key, contract_version, agent_version,
		           capabilities, identity_fingerprint, status, last_seen_at, last_heartbeat_at, created_at`,
		in.TenantID,
		in.NodeID,
		in.NodeKeyID,
		in.ContractVersion,
		in.AgentVersion,
	)
	item, err := scanNode(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Node{}, sql.ErrNoRows
		}
		return model.Node{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) InsertNodeHeartbeatEvent(ctx context.Context, nodeID string, tenantID string, status string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	blob, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal node heartbeat metadata: %w", err)
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO node_heartbeats (node_id, tenant_id, status, metadata) VALUES ($1, $2, $3, $4::jsonb)`,
		nodeID,
		tenantID,
		status,
		string(blob),
	)
	if err != nil {
		return mapDBError(err)
	}
	return nil
}

func scanTenantPolicy(scan func(dest ...any) error) (model.TenantPolicy, error) {
	var out model.TenantPolicy
	var trafficQuota sql.NullInt64
	var deviceLimit sql.NullInt64
	var defaultTTL sql.NullInt64
	err := scan(&out.TenantID, &trafficQuota, &deviceLimit, &defaultTTL, &out.UpdatedBy, &out.UpdatedAt)
	if err != nil {
		return model.TenantPolicy{}, err
	}
	out.TrafficQuotaBytes = int64PtrFromNull(trafficQuota)
	out.DeviceLimit = intPtrFromNull(deviceLimit)
	out.DefaultKeyTTLSeconds = intPtrFromNull(defaultTTL)
	return out, nil
}

func scanUserPolicyOverride(scan func(dest ...any) error) (model.UserPolicyOverride, error) {
	var out model.UserPolicyOverride
	var trafficQuota sql.NullInt64
	var deviceLimit sql.NullInt64
	var keyTTL sql.NullInt64
	err := scan(&out.TenantID, &out.UserID, &trafficQuota, &deviceLimit, &keyTTL, &out.UpdatedBy, &out.UpdatedAt)
	if err != nil {
		return model.UserPolicyOverride{}, err
	}
	out.TrafficQuotaBytes = int64PtrFromNull(trafficQuota)
	out.DeviceLimit = intPtrFromNull(deviceLimit)
	out.KeyTTLSeconds = intPtrFromNull(keyTTL)
	return out, nil
}

func scanNode(scan func(dest ...any) error) (model.Node, error) {
	var out model.Node
	var capabilitiesRaw []byte
	var lastSeenAt sql.NullTime
	var lastHeartbeatAt sql.NullTime
	err := scan(
		&out.ID,
		&out.TenantID,
		&out.Region,
		&out.Hostname,
		&out.NodeKeyID,
		&out.NodePublicKey,
		&out.ContractVersion,
		&out.AgentVersion,
		&capabilitiesRaw,
		&out.IdentityFingerprint,
		&out.Status,
		&lastSeenAt,
		&lastHeartbeatAt,
		&out.CreatedAt,
	)
	if err != nil {
		return model.Node{}, err
	}
	if len(capabilitiesRaw) > 0 {
		if err := json.Unmarshal(capabilitiesRaw, &out.Capabilities); err != nil {
			return model.Node{}, fmt.Errorf("decode node capabilities: %w", err)
		}
	}
	if lastSeenAt.Valid {
		t := lastSeenAt.Time
		out.LastSeenAt = &t
	}
	if lastHeartbeatAt.Valid {
		t := lastHeartbeatAt.Time
		out.LastHeartbeatAt = &t
	}
	return out, nil
}

func intPtrFromNull(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func int64PtrFromNull(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("db operation failed: %w", err)
	}

	switch pgErr.Code {
	case "23505":
		return &DBError{Kind: ErrorKindConflict, Constraint: pgErr.ConstraintName, Message: "unique constraint violation", Err: err}
	case "23503":
		return &DBError{Kind: ErrorKindForeignKey, Constraint: pgErr.ConstraintName, Message: "foreign key violation", Err: err}
	case "23514", "23502", "22P02", "22001":
		return &DBError{Kind: ErrorKindValidation, Constraint: pgErr.ConstraintName, Message: normalizeValidationMessage(pgErr), Err: err}
	default:
		return fmt.Errorf("db operation failed: %w", err)
	}
}

func normalizeValidationMessage(pgErr *pgconn.PgError) string {
	if msg := validationConstraintMessage(pgErr.ConstraintName); msg != "" {
		return msg
	}
	msg := strings.TrimSpace(pgErr.Message)
	if msg == "" {
		return "validation failed"
	}
	return msg
}

func validationConstraintMessage(constraint string) string {
	switch strings.TrimSpace(constraint) {
	case "chk_tenant_policies_traffic_quota_non_negative", "chk_user_policy_overrides_traffic_quota_non_negative":
		return "traffic_quota_bytes must be >= 0"
	case "chk_tenant_policies_device_limit_positive", "chk_user_policy_overrides_device_limit_positive":
		return "device_limit must be > 0"
	case "chk_tenant_policies_default_ttl_positive", "chk_user_policy_overrides_ttl_positive":
		return "ttl must be > 0"
	case "chk_nodes_status_stage5":
		return "invalid node status"
	default:
		return ""
	}
}

func normalizePaging(limit int, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
