package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type NodeCertificateStore interface {
	IssueNodeCertificate(ctx context.Context, in model.IssueNodeCertificateRequest) (model.NodeCertificate, error)
	GetActiveNodeCertificate(ctx context.Context, tenantID string, nodeID string) (model.NodeCertificate, error)
	GetNodeCertificateBySerial(ctx context.Context, tenantID string, nodeID string, serialNumber string) (model.NodeCertificate, error)
	GetNodeCertificateByID(ctx context.Context, tenantID string, nodeID string, certificateID string) (model.NodeCertificate, error)
	ListNodeCertificates(ctx context.Context, q model.ListNodeCertificatesQuery) ([]model.NodeCertificate, error)
	ListNodeCertificatesExpiringBefore(ctx context.Context, before time.Time, limit int) ([]model.NodeCertificate, error)
	RevokeNodeCertificateByID(ctx context.Context, tenantID string, nodeID string, certificateID string) (model.NodeCertificate, error)
	RevokeNodeCertificateBySerial(ctx context.Context, tenantID string, nodeID string, serialNumber string) (model.NodeCertificate, error)
}

func (s *SQLStore) IssueNodeCertificate(ctx context.Context, in model.IssueNodeCertificateRequest) (model.NodeCertificate, error) {
	var rotateFrom any
	if in.RotateFromCertID != nil {
		rotateFrom = *in.RotateFromCertID
	}
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO node_certificates (
		   tenant_id, node_id, serial_number, cert_pem, ca_id, issuer_id, issuer, not_before, not_after, rotate_from_cert_id
		 ) VALUES (
		   $1, $2, $3, $4, $5, NULLIF($6,''), $7, $8, $9, NULLIF($10,'')::uuid
		 )
		 RETURNING id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at`,
		in.TenantID,
		in.NodeID,
		in.SerialNumber,
		in.CertPEM,
		in.CAID,
		in.IssuerID,
		in.Issuer,
		in.NotBefore,
		in.NotAfter,
		rotateFrom,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) GetActiveNodeCertificate(ctx context.Context, tenantID string, nodeID string) (model.NodeCertificate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at
		 FROM node_certificates
		 WHERE tenant_id = $1
		   AND node_id = $2
		   AND revoked_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		tenantID,
		nodeID,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, fmt.Errorf("get active node certificate: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetNodeCertificateBySerial(ctx context.Context, tenantID string, nodeID string, serialNumber string) (model.NodeCertificate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at
		 FROM node_certificates
		 WHERE tenant_id = $1
		   AND node_id = $2
		   AND serial_number = $3`,
		tenantID,
		nodeID,
		serialNumber,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, fmt.Errorf("get node certificate by serial: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetNodeCertificateByID(ctx context.Context, tenantID string, nodeID string, certificateID string) (model.NodeCertificate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at
		 FROM node_certificates
		 WHERE tenant_id = $1
		   AND node_id = $2
		   AND id = $3`,
		tenantID,
		nodeID,
		certificateID,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, fmt.Errorf("get node certificate by id: %w", err)
	}
	return item, nil
}

func (s *SQLStore) ListNodeCertificates(ctx context.Context, q model.ListNodeCertificatesQuery) ([]model.NodeCertificate, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	status := strings.ToLower(strings.TrimSpace(q.Status))
	base := `SELECT id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at
	         FROM node_certificates
	         WHERE tenant_id = $1 AND node_id = $2`
	args := []any{q.TenantID, q.NodeID}
	switch status {
	case "active":
		base += ` AND revoked_at IS NULL`
	case "revoked":
		base += ` AND revoked_at IS NOT NULL`
	}
	base += ` ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("list node certificates: %w", err)
	}
	defer rows.Close()

	items := make([]model.NodeCertificate, 0)
	for rows.Next() {
		item, scanErr := scanNodeCertificate(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan node certificate: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node certificates: %w", err)
	}
	return items, nil
}

func (s *SQLStore) ListNodeCertificatesExpiringBefore(ctx context.Context, before time.Time, limit int) ([]model.NodeCertificate, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at
		 FROM node_certificates
		 WHERE revoked_at IS NULL
		   AND not_after <= $1
		 ORDER BY not_after ASC
		 LIMIT $2`,
		before,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list expiring node certificates: %w", err)
	}
	defer rows.Close()

	items := make([]model.NodeCertificate, 0)
	for rows.Next() {
		item, scanErr := scanNodeCertificate(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan expiring node certificate: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expiring node certificates: %w", err)
	}
	return items, nil
}

func (s *SQLStore) RevokeNodeCertificateByID(ctx context.Context, tenantID string, nodeID string, certificateID string) (model.NodeCertificate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`UPDATE node_certificates
		 SET revoked_at = NOW()
		 WHERE tenant_id = $1
		   AND node_id = $2
		   AND id = $3
		   AND revoked_at IS NULL
		 RETURNING id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at`,
		tenantID,
		nodeID,
		certificateID,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, mapDBError(err)
	}
	return item, nil
}

func (s *SQLStore) RevokeNodeCertificateBySerial(ctx context.Context, tenantID string, nodeID string, serialNumber string) (model.NodeCertificate, error) {
	row := s.db.QueryRowContext(
		ctx,
		`UPDATE node_certificates
		 SET revoked_at = NOW()
		 WHERE tenant_id = $1
		   AND node_id = $2
		   AND serial_number = $3
		   AND revoked_at IS NULL
		 RETURNING id::text, tenant_id::text, node_id::text, serial_number, cert_pem, ca_id, COALESCE(issuer_id,''), issuer, not_before, not_after, revoked_at, rotate_from_cert_id::text, created_at`,
		tenantID,
		nodeID,
		serialNumber,
	)
	item, err := scanNodeCertificate(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, sql.ErrNoRows
		}
		return model.NodeCertificate{}, mapDBError(err)
	}
	return item, nil
}

func scanNodeCertificate(scan func(dest ...any) error) (model.NodeCertificate, error) {
	var out model.NodeCertificate
	var revokedAt sql.NullTime
	var rotateFrom sql.NullString
	err := scan(
		&out.ID,
		&out.TenantID,
		&out.NodeID,
		&out.SerialNumber,
		&out.CertPEM,
		&out.CAID,
		&out.IssuerID,
		&out.Issuer,
		&out.NotBefore,
		&out.NotAfter,
		&revokedAt,
		&rotateFrom,
		&out.CreatedAt,
	)
	if err != nil {
		return model.NodeCertificate{}, err
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		out.RevokedAt = &t
	}
	if rotateFrom.Valid {
		v := rotateFrom.String
		out.RotateFromCertID = &v
	}
	return out, nil
}
