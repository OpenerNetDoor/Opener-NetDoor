package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

func (s *SQLStore) CountUsersByStatus(ctx context.Context, tenantID string, status string) (int, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM users
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ($2 = '' OR status = $2)`,
		strings.TrimSpace(tenantID),
		strings.TrimSpace(status),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users by status: %w", err)
	}
	return count, nil
}

func (s *SQLStore) CountUsersCreatedBefore(ctx context.Context, tenantID string, before time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM users
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND created_at < $2`,
		strings.TrimSpace(tenantID),
		before.UTC(),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users before: %w", err)
	}
	return count, nil
}

func (s *SQLStore) CountAccessKeysByStatus(ctx context.Context, tenantID string, status string) (int, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM access_keys
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ($2 = '' OR status = $2)`,
		strings.TrimSpace(tenantID),
		strings.TrimSpace(status),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count access keys by status: %w", err)
	}
	return count, nil
}

func (s *SQLStore) ListTrafficUsageSeries(ctx context.Context, tenantID string, since time.Time, until time.Time) ([]model.OpsTrafficPoint, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT date_trunc('day', ts_hour) AS ts_day,
		        COALESCE(SUM(bytes_in), 0) AS bytes_in,
		        COALESCE(SUM(bytes_out), 0) AS bytes_out,
		        COALESCE(SUM(bytes_in + bytes_out), 0) AS bytes_total
		 FROM traffic_usage_hourly
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ts_hour >= $2
		   AND ts_hour < $3
		 GROUP BY ts_day
		 ORDER BY ts_day`,
		strings.TrimSpace(tenantID),
		since.UTC(),
		until.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("query traffic usage series: %w", err)
	}
	defer rows.Close()

	out := make([]model.OpsTrafficPoint, 0)
	for rows.Next() {
		var item model.OpsTrafficPoint
		if err := rows.Scan(&item.TsHour, &item.BytesIn, &item.BytesOut, &item.BytesTotal); err != nil {
			return nil, fmt.Errorf("scan traffic usage series: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate traffic usage series: %w", err)
	}
	return out, nil
}

func (s *SQLStore) ListUserGrowthByDay(ctx context.Context, tenantID string, since time.Time, until time.Time) ([]model.OpsUserGrowthPoint, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
		        COUNT(*)::int AS new_users
		 FROM users
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND created_at >= $2
		   AND created_at < $3
		 GROUP BY day
		 ORDER BY day`,
		strings.TrimSpace(tenantID),
		since.UTC(),
		until.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("query user growth by day: %w", err)
	}
	defer rows.Close()

	out := make([]model.OpsUserGrowthPoint, 0)
	for rows.Next() {
		var item model.OpsUserGrowthPoint
		if err := rows.Scan(&item.Day, &item.NewUsers); err != nil {
			return nil, fmt.Errorf("scan user growth by day: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user growth by day: %w", err)
	}
	return out, nil
}

func (s *SQLStore) ListProtocolUsageBetween(ctx context.Context, tenantID string, since time.Time, until time.Time) ([]model.OpsProtocolUsagePoint, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT protocol, COALESCE(SUM(bytes_in + bytes_out), 0) AS bytes_total
		 FROM traffic_usage_hourly
		 WHERE ($1 = '' OR tenant_id::text = $1)
		   AND ts_hour >= $2
		   AND ts_hour < $3
		 GROUP BY protocol
		 ORDER BY bytes_total DESC, protocol ASC`,
		strings.TrimSpace(tenantID),
		since.UTC(),
		until.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("query protocol usage: %w", err)
	}
	defer rows.Close()

	out := make([]model.OpsProtocolUsagePoint, 0)
	for rows.Next() {
		var item model.OpsProtocolUsagePoint
		if err := rows.Scan(&item.Protocol, &item.BytesTotal); err != nil {
			return nil, fmt.Errorf("scan protocol usage: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate protocol usage: %w", err)
	}
	return out, nil
}

func (s *SQLStore) ListTopServersByTraffic(ctx context.Context, tenantID string, since time.Time, until time.Time, limit int) ([]model.OpsTopServerPoint, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT n.id::text, n.hostname, n.region,
		        COALESCE(SUM(t.bytes_in + t.bytes_out), 0) AS bytes_total
		 FROM nodes n
		 JOIN traffic_usage_hourly t ON t.node_id = n.id
		 WHERE ($1 = '' OR n.tenant_id::text = $1)
		   AND t.ts_hour >= $2
		   AND t.ts_hour < $3
		 GROUP BY n.id, n.hostname, n.region
		 HAVING COALESCE(SUM(t.bytes_in + t.bytes_out), 0) > 0
		 ORDER BY bytes_total DESC, n.hostname ASC
		 LIMIT $4`,
		strings.TrimSpace(tenantID),
		since.UTC(),
		until.UTC(),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query top servers by traffic: %w", err)
	}
	defer rows.Close()

	out := make([]model.OpsTopServerPoint, 0)
	for rows.Next() {
		var item model.OpsTopServerPoint
		if err := rows.Scan(&item.NodeID, &item.Hostname, &item.Region, &item.BytesTotal); err != nil {
			return nil, fmt.Errorf("scan top servers by traffic: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top servers by traffic: %w", err)
	}
	return out, nil
}
