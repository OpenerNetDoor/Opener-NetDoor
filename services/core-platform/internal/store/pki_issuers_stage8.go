package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type PKIIssuerStore interface {
	CreatePKIIssuer(ctx context.Context, in model.CreatePKIIssuerRequest) (model.PKIIssuer, error)
	ListPKIIssuers(ctx context.Context, q model.ListPKIIssuersQuery) ([]model.PKIIssuer, error)
	GetPKIIssuerByIssuerID(ctx context.Context, issuerID string) (model.PKIIssuer, error)
	GetPKIIssuerByCAID(ctx context.Context, caID string) (model.PKIIssuer, error)
	GetActivePKIIssuer(ctx context.Context) (model.PKIIssuer, error)
	ActivatePKIIssuer(ctx context.Context, issuerID string) (model.CARotationResult, error)
	RetirePKIIssuer(ctx context.Context, issuerID string) (model.PKIIssuer, error)
}

func (s *SQLStore) CreatePKIIssuer(ctx context.Context, in model.CreatePKIIssuerRequest) (model.PKIIssuer, error) {
	if strings.TrimSpace(in.Source) == "" {
		in.Source = "file"
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	blob, err := json.Marshal(metadata)
	if err != nil {
		return model.PKIIssuer{}, fmt.Errorf("marshal issuer metadata: %w", err)
	}
	row := s.db.QueryRowContext(
		ctx,
		`INSERT INTO pki_issuers (issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, metadata)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6::jsonb)
		 RETURNING id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at`,
		strings.TrimSpace(in.IssuerID),
		strings.TrimSpace(in.Source),
		strings.TrimSpace(in.CAID),
		strings.TrimSpace(in.IssuerName),
		strings.TrimSpace(in.CACertPEM),
		string(blob),
	)
	item, scanErr := scanPKIIssuer(row.Scan)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return model.PKIIssuer{}, sql.ErrNoRows
		}
		return model.PKIIssuer{}, mapDBError(scanErr)
	}
	return item, nil
}

func (s *SQLStore) ListPKIIssuers(ctx context.Context, q model.ListPKIIssuersQuery) ([]model.PKIIssuer, error) {
	limit, offset := normalizePaging(q.Limit, q.Offset)
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at
		 FROM pki_issuers
		 WHERE ($1 = '' OR source = $1)
		   AND ($2 = '' OR status = $2)
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		strings.TrimSpace(q.Source),
		strings.TrimSpace(q.Status),
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list pki issuers: %w", err)
	}
	defer rows.Close()

	items := make([]model.PKIIssuer, 0)
	for rows.Next() {
		item, scanErr := scanPKIIssuer(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan pki issuer: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pki issuers: %w", err)
	}
	return items, nil
}

func (s *SQLStore) GetPKIIssuerByIssuerID(ctx context.Context, issuerID string) (model.PKIIssuer, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at
		 FROM pki_issuers
		 WHERE issuer_id = $1`,
		strings.TrimSpace(issuerID),
	)
	item, err := scanPKIIssuer(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PKIIssuer{}, sql.ErrNoRows
		}
		return model.PKIIssuer{}, fmt.Errorf("get pki issuer by issuer_id: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetPKIIssuerByCAID(ctx context.Context, caID string) (model.PKIIssuer, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at
		 FROM pki_issuers
		 WHERE ca_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		strings.TrimSpace(caID),
	)
	item, err := scanPKIIssuer(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PKIIssuer{}, sql.ErrNoRows
		}
		return model.PKIIssuer{}, fmt.Errorf("get pki issuer by ca_id: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetActivePKIIssuer(ctx context.Context) (model.PKIIssuer, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at
		 FROM pki_issuers
		 WHERE status = 'active'
		 ORDER BY activated_at DESC NULLS LAST, created_at DESC
		 LIMIT 1`,
	)
	item, err := scanPKIIssuer(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PKIIssuer{}, sql.ErrNoRows
		}
		return model.PKIIssuer{}, fmt.Errorf("get active pki issuer: %w", err)
	}
	return item, nil
}

func (s *SQLStore) ActivatePKIIssuer(ctx context.Context, issuerID string) (model.CARotationResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.CARotationResult{}, fmt.Errorf("begin activate issuer tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var previous *model.PKIIssuer
	var previousIssuerID string
	currentRow := tx.QueryRowContext(
		ctx,
		`SELECT id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at
		 FROM pki_issuers
		 WHERE status = 'active'
		 FOR UPDATE`,
	)
	current, currentErr := scanPKIIssuer(currentRow.Scan)
	if currentErr == nil {
		if current.IssuerID == strings.TrimSpace(issuerID) {
			result := model.CARotationResult{ActiveIssuer: current, RotatedAt: time.Now().UTC()}
			if commitErr := tx.Commit(); commitErr != nil {
				return model.CARotationResult{}, fmt.Errorf("commit activate issuer tx: %w", commitErr)
			}
			return result, nil
		}
		previousIssuerID = current.IssuerID
		retireRow := tx.QueryRowContext(
			ctx,
			`UPDATE pki_issuers
			 SET status = 'retired', retired_at = NOW()
			 WHERE issuer_id = $1
			 RETURNING id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at`,
			current.IssuerID,
		)
		retired, retireErr := scanPKIIssuer(retireRow.Scan)
		if retireErr != nil {
			return model.CARotationResult{}, mapDBError(retireErr)
		}
		previous = &retired
	} else if !errors.Is(currentErr, sql.ErrNoRows) {
		return model.CARotationResult{}, fmt.Errorf("load active issuer: %w", currentErr)
	}

	activeRow := tx.QueryRowContext(
		ctx,
		`UPDATE pki_issuers
		 SET status = 'active', activated_at = NOW(), retired_at = NULL, rotate_from_issuer_id = NULLIF($2, '')
		 WHERE issuer_id = $1
		 RETURNING id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at`,
		strings.TrimSpace(issuerID),
		previousIssuerID,
	)
	active, activeErr := scanPKIIssuer(activeRow.Scan)
	if activeErr != nil {
		if errors.Is(activeErr, sql.ErrNoRows) {
			return model.CARotationResult{}, sql.ErrNoRows
		}
		return model.CARotationResult{}, mapDBError(activeErr)
	}

	if err := tx.Commit(); err != nil {
		return model.CARotationResult{}, fmt.Errorf("commit activate issuer tx: %w", err)
	}
	return model.CARotationResult{ActiveIssuer: active, PreviousIssuer: previous, RotatedAt: time.Now().UTC()}, nil
}

func (s *SQLStore) RetirePKIIssuer(ctx context.Context, issuerID string) (model.PKIIssuer, error) {
	row := s.db.QueryRowContext(
		ctx,
		`UPDATE pki_issuers
		 SET status = 'retired', retired_at = NOW()
		 WHERE issuer_id = $1
		 RETURNING id::text, issuer_id, source, ca_id, issuer_name, ca_cert_pem, status, activated_at, retired_at, rotate_from_issuer_id, metadata, created_at`,
		strings.TrimSpace(issuerID),
	)
	item, err := scanPKIIssuer(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PKIIssuer{}, sql.ErrNoRows
		}
		return model.PKIIssuer{}, mapDBError(err)
	}
	return item, nil
}

func scanPKIIssuer(scan func(dest ...any) error) (model.PKIIssuer, error) {
	var out model.PKIIssuer
	var activatedAt sql.NullTime
	var retiredAt sql.NullTime
	var rotateFrom sql.NullString
	var metadataRaw []byte
	err := scan(
		&out.ID,
		&out.IssuerID,
		&out.Source,
		&out.CAID,
		&out.IssuerName,
		&out.CACertPEM,
		&out.Status,
		&activatedAt,
		&retiredAt,
		&rotateFrom,
		&metadataRaw,
		&out.CreatedAt,
	)
	if err != nil {
		return model.PKIIssuer{}, err
	}
	if activatedAt.Valid {
		t := activatedAt.Time
		out.ActivatedAt = &t
	}
	if retiredAt.Valid {
		t := retiredAt.Time
		out.RetiredAt = &t
	}
	if rotateFrom.Valid {
		v := rotateFrom.String
		out.RotateFromIssuerID = &v
	}
	if len(metadataRaw) > 0 {
		if err := json.Unmarshal(metadataRaw, &out.Metadata); err != nil {
			return model.PKIIssuer{}, fmt.Errorf("decode issuer metadata: %w", err)
		}
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out, nil
}
