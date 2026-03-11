package service

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
)

func (s *CoreService) pkiIssuerStore() (store.PKIIssuerStore, error) {
	ps, ok := s.store.(store.PKIIssuerStore)
	if !ok {
		return nil, &AppError{Status: 500, Code: "node_pki_unavailable", Message: "pki issuer store is not configured"}
	}
	return ps, nil
}

func (s *CoreService) ensureSigningIssuer(ctx context.Context) (model.PKIIssuer, error) {
	ps, err := s.pkiIssuerStore()
	if err != nil {
		return model.PKIIssuer{}, err
	}
	issuerID := strings.TrimSpace(s.opts.NodeCAActiveIssuerID)
	if issuerID == "" {
		issuerID = "default-file-issuer"
	}

	signer, err := s.nodeCertificateSigner()
	if err != nil {
		return model.PKIIssuer{}, err
	}
	if signer.Mode() == "external" {
		externalIssuer, getErr := ps.GetPKIIssuerByIssuerID(ctx, issuerID)
		if getErr != nil {
			if errors.Is(getErr, sql.ErrNoRows) {
				return model.PKIIssuer{}, &AppError{Status: 503, Code: "active_issuer_not_found", Message: "active external issuer metadata is not configured", Err: getErr}
			}
			return model.PKIIssuer{}, mapStoreError("issuer_lookup_failed", getErr)
		}
		if _, activateErr := ps.ActivatePKIIssuer(ctx, issuerID); activateErr != nil && !errors.Is(activateErr, sql.ErrNoRows) {
			return model.PKIIssuer{}, mapStoreError("issuer_activate_failed", activateErr)
		}
		return externalIssuer, nil
	}

	if err := s.ensureCA(); err != nil {
		return model.PKIIssuer{}, err
	}
	if s.caBundle == nil || s.caBundle.Cert == nil {
		return model.PKIIssuer{}, &AppError{Status: 500, Code: "node_pki_init_failed", Message: "missing CA bundle"}
	}

	currentIssuer, getErr := ps.GetPKIIssuerByIssuerID(ctx, issuerID)
	if getErr != nil {
		if !errors.Is(getErr, sql.ErrNoRows) {
			return model.PKIIssuer{}, mapStoreError("issuer_lookup_failed", getErr)
		}

		byCA, byCAErr := ps.GetPKIIssuerByCAID(ctx, s.caBundle.ID)
		if byCAErr == nil {
			issuerID = byCA.IssuerID
			currentIssuer = byCA
		} else if !errors.Is(byCAErr, sql.ErrNoRows) {
			return model.PKIIssuer{}, mapStoreError("issuer_lookup_failed", byCAErr)
		} else {
			caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.caBundle.Cert.Raw}))
			created, createErr := ps.CreatePKIIssuer(ctx, model.CreatePKIIssuerRequest{
				IssuerID:   issuerID,
				Source:     "file",
				CAID:       s.caBundle.ID,
				IssuerName: s.caBundle.Issuer,
				CACertPEM:  caPEM,
				Metadata: map[string]any{
					"mode": "file",
				},
			})
			if createErr != nil {
				return model.PKIIssuer{}, mapStoreError("issuer_create_failed", createErr)
			}
			currentIssuer = created
		}
	}

	active, activeErr := ps.GetActivePKIIssuer(ctx)
	if activeErr != nil {
		if errors.Is(activeErr, sql.ErrNoRows) {
			rotation, activateErr := ps.ActivatePKIIssuer(ctx, issuerID)
			if activateErr != nil {
				return model.PKIIssuer{}, mapStoreError("issuer_activate_failed", activateErr)
			}
			return rotation.ActiveIssuer, nil
		}
		return model.PKIIssuer{}, mapStoreError("issuer_lookup_failed", activeErr)
	}
	if active.IssuerID != issuerID {
		rotation, activateErr := ps.ActivatePKIIssuer(ctx, issuerID)
		if activateErr != nil {
			return model.PKIIssuer{}, mapStoreError("issuer_activate_failed", activateErr)
		}
		return rotation.ActiveIssuer, nil
	}
	if currentIssuer.IssuerID != "" {
		return currentIssuer, nil
	}
	return active, nil
}

func (s *CoreService) validateIssuerTrust(ctx context.Context, cert model.NodeCertificate) error {
	ps, err := s.pkiIssuerStore()
	if err != nil {
		return err
	}

	issuerID := strings.TrimSpace(cert.IssuerID)
	if issuerID == "" {
		issuer, issuerErr := ps.GetPKIIssuerByCAID(ctx, cert.CAID)
		if issuerErr != nil {
			if errors.Is(issuerErr, sql.ErrNoRows) {
				return &AppError{Status: 401, Code: "node_certificate_untrusted_issuer", Message: "issuer is not trusted", Err: issuerErr}
			}
			return mapStoreError("node_certificate_verify_failed", issuerErr)
		}
		issuerID = issuer.IssuerID
	}

	trusted := make(map[string]struct{})
	if id := strings.TrimSpace(s.opts.NodeCAActiveIssuerID); id != "" {
		trusted[id] = struct{}{}
	}
	for _, id := range s.opts.NodeCAPreviousIssuerIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		trusted[id] = struct{}{}
	}

	if active, activeErr := ps.GetActivePKIIssuer(ctx); activeErr == nil {
		trusted[active.IssuerID] = struct{}{}
	}

	if _, ok := trusted[issuerID]; ok {
		return nil
	}
	return &AppError{Status: 401, Code: "node_certificate_untrusted_issuer", Message: "issuer is not trusted for current verification window"}
}

func (s *CoreService) trustedRootPool(ctx context.Context, cert model.NodeCertificate) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	added := false
	addIssuer := func(issuer model.PKIIssuer) {
		if strings.TrimSpace(issuer.CACertPEM) == "" {
			return
		}
		block, _ := pem.Decode([]byte(issuer.CACertPEM))
		if block == nil {
			return
		}
		parsed, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return
		}
		pool.AddCert(parsed)
		added = true
	}

	ps, err := s.pkiIssuerStore()
	if err == nil {
		issuerIDs := make(map[string]struct{})
		if id := strings.TrimSpace(cert.IssuerID); id != "" {
			issuerIDs[id] = struct{}{}
		}
		if id := strings.TrimSpace(s.opts.NodeCAActiveIssuerID); id != "" {
			issuerIDs[id] = struct{}{}
		}
		for _, id := range s.opts.NodeCAPreviousIssuerIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				issuerIDs[id] = struct{}{}
			}
		}
		if active, activeErr := ps.GetActivePKIIssuer(ctx); activeErr == nil {
			issuerIDs[active.IssuerID] = struct{}{}
		}
		for issuerID := range issuerIDs {
			issuer, issuerErr := ps.GetPKIIssuerByIssuerID(ctx, issuerID)
			if issuerErr == nil {
				addIssuer(issuer)
			}
		}
		if !added && strings.TrimSpace(cert.CAID) != "" {
			issuer, issuerErr := ps.GetPKIIssuerByCAID(ctx, cert.CAID)
			if issuerErr == nil {
				addIssuer(issuer)
			}
		}
	}

	if !added && s.caBundle != nil && s.caBundle.Cert != nil {
		pool.AddCert(s.caBundle.Cert)
		added = true
	}
	if !added {
		return nil, &AppError{Status: 500, Code: "node_certificate_verify_failed", Message: "no trusted CA roots available"}
	}
	return pool, nil
}

func (s *CoreService) ListPKIIssuers(ctx context.Context, actor model.ActorPrincipal, q model.ListPKIIssuersQuery) ([]model.PKIIssuer, error) {
	if !actor.IsPlatformAdmin() {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	ps, err := s.pkiIssuerStore()
	if err != nil {
		return nil, err
	}
	items, err := ps.ListPKIIssuers(ctx, q)
	if err != nil {
		return nil, mapStoreError("issuer_list_failed", err)
	}
	return items, nil
}

func (s *CoreService) CreatePKIIssuer(ctx context.Context, actor model.ActorPrincipal, in model.CreatePKIIssuerRequest) (model.PKIIssuer, error) {
	if !actor.IsPlatformAdmin() {
		return model.PKIIssuer{}, &AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	if strings.TrimSpace(in.IssuerID) == "" {
		in.IssuerID = s.opts.NodeCAActiveIssuerID
	}
	if strings.TrimSpace(in.Source) == "" {
		in.Source = s.opts.NodeCAMode
	}
	in.Source = strings.ToLower(strings.TrimSpace(in.Source))
	if in.Source != "file" && in.Source != "external" {
		return model.PKIIssuer{}, &AppError{Status: 400, Code: "validation_error", Message: "source must be file or external"}
	}

	if in.Source == "file" {
		if err := s.ensureCA(); err != nil {
			return model.PKIIssuer{}, err
		}
		if s.caBundle == nil || s.caBundle.Cert == nil {
			return model.PKIIssuer{}, &AppError{Status: 500, Code: "node_pki_init_failed", Message: "missing CA bundle"}
		}
		if strings.TrimSpace(in.CAID) == "" {
			in.CAID = s.caBundle.ID
		}
		if strings.TrimSpace(in.IssuerName) == "" {
			in.IssuerName = s.caBundle.Issuer
		}
		if strings.TrimSpace(in.CACertPEM) == "" {
			in.CACertPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.caBundle.Cert.Raw}))
		}
	} else {
		if strings.TrimSpace(in.CACertPEM) == "" {
			return model.PKIIssuer{}, &AppError{Status: 400, Code: "validation_error", Message: "ca_cert_pem is required for external issuer metadata"}
		}
		if strings.TrimSpace(in.CAID) == "" {
			in.CAID = hashString(in.CACertPEM)
		}
		if strings.TrimSpace(in.IssuerName) == "" {
			in.IssuerName = in.IssuerID
		}
	}

	ps, err := s.pkiIssuerStore()
	if err != nil {
		return model.PKIIssuer{}, err
	}

	item, getErr := ps.GetPKIIssuerByIssuerID(ctx, in.IssuerID)
	if getErr != nil {
		if !errors.Is(getErr, sql.ErrNoRows) {
			return model.PKIIssuer{}, mapStoreError("issuer_get_failed", getErr)
		}
		item, err = ps.CreatePKIIssuer(ctx, in)
		if err != nil {
			return model.PKIIssuer{}, mapStoreError("issuer_create_failed", err)
		}
		if auditErr := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
			ActorType:  "admin",
			ActorSub:   actor.Subject,
			Action:     "issuer_create",
			TargetType: "pki_issuer",
			Metadata: map[string]any{
				"issuer_id": item.IssuerID,
				"source":    item.Source,
			},
			OccurredAt: time.Now().UTC(),
		}); auditErr != nil {
			return model.PKIIssuer{}, mapStoreError("issuer_create_failed", auditErr)
		}
	}

	if in.Activate {
		rotation, activateErr := ps.ActivatePKIIssuer(ctx, item.IssuerID)
		if activateErr != nil {
			return model.PKIIssuer{}, mapStoreError("issuer_activate_failed", activateErr)
		}
		if rotation.PreviousIssuer != nil {
			s.auditBestEffort(ctx, model.AuditLogEvent{
				ActorType:  "admin",
				ActorSub:   actor.Subject,
				Action:     "issuer_retire",
				TargetType: "pki_issuer",
				Metadata:   map[string]any{"issuer_id": rotation.PreviousIssuer.IssuerID},
				OccurredAt: time.Now().UTC(),
			})
		}
		s.auditBestEffort(ctx, model.AuditLogEvent{
			ActorType:  "admin",
			ActorSub:   actor.Subject,
			Action:     "issuer_activate",
			TargetType: "pki_issuer",
			Metadata:   map[string]any{"issuer_id": rotation.ActiveIssuer.IssuerID},
			OccurredAt: time.Now().UTC(),
		})
		return rotation.ActiveIssuer, nil
	}

	return item, nil
}

func (s *CoreService) ActivatePKIIssuer(ctx context.Context, actor model.ActorPrincipal, in model.ActivatePKIIssuerRequest) (model.CARotationResult, error) {
	if !actor.IsPlatformAdmin() {
		return model.CARotationResult{}, &AppError{Status: 403, Code: "forbidden", Message: "only platform admin can manage pki issuers"}
	}
	if strings.TrimSpace(in.IssuerID) == "" {
		return model.CARotationResult{}, &AppError{Status: 400, Code: "validation_error", Message: "issuer_id is required"}
	}
	ps, err := s.pkiIssuerStore()
	if err != nil {
		return model.CARotationResult{}, err
	}
	result, err := ps.ActivatePKIIssuer(ctx, in.IssuerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CARotationResult{}, &AppError{Status: 404, Code: "issuer_not_found", Message: "issuer not found", Err: err}
		}
		return model.CARotationResult{}, mapStoreError("issuer_activate_failed", err)
	}
	s.auditBestEffort(ctx, model.AuditLogEvent{
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "issuer_activate",
		TargetType: "pki_issuer",
		Metadata:   map[string]any{"issuer_id": result.ActiveIssuer.IssuerID},
		OccurredAt: time.Now().UTC(),
	})
	if result.PreviousIssuer != nil {
		s.auditBestEffort(ctx, model.AuditLogEvent{
			ActorType:  "admin",
			ActorSub:   actor.Subject,
			Action:     "issuer_retire",
			TargetType: "pki_issuer",
			Metadata:   map[string]any{"issuer_id": result.PreviousIssuer.IssuerID},
			OccurredAt: time.Now().UTC(),
		})
	}
	return result, nil
}

func (s *CoreService) RenewNodeCertificate(ctx context.Context, actor model.ActorPrincipal, in model.RenewNodeCertificateRequest) (model.RenewNodeCertificateResult, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.NodeID) == "" {
		return model.RenewNodeCertificateResult{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.RenewNodeCertificateResult{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}

	node, err := s.store.GetNodeByID(ctx, in.TenantID, in.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RenewNodeCertificateResult{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.RenewNodeCertificateResult{}, &AppError{Status: 500, Code: "node_certificate_renew_failed", Message: "failed to load node", Err: err}
	}

	cs, err := s.nodeCertStore()
	if err != nil {
		return model.RenewNodeCertificateResult{}, err
	}
	active, getErr := cs.GetActiveNodeCertificate(ctx, in.TenantID, in.NodeID)
	if getErr != nil {
		if !errors.Is(getErr, sql.ErrNoRows) {
			return model.RenewNodeCertificateResult{}, mapStoreError("node_certificate_renew_failed", getErr)
		}
		bundle, issueErr := s.issueNodeCertificateInternal(ctx, node, nil)
		if issueErr != nil {
			return model.RenewNodeCertificateResult{}, issueErr
		}
		if bundle == nil {
			return model.RenewNodeCertificateResult{}, &AppError{Status: 500, Code: "node_certificate_renew_failed", Message: "certificate renewal returned empty result"}
		}
		s.auditBestEffort(ctx, model.AuditLogEvent{
			TenantID:   node.TenantID,
			ActorType:  "admin",
			ActorSub:   actor.Subject,
			Action:     "cert_renew",
			TargetType: "node",
			TargetID:   node.ID,
			Metadata:   map[string]any{"certificate_id": bundle.Certificate.ID, "serial_number": bundle.Certificate.SerialNumber},
			OccurredAt: time.Now().UTC(),
		})
		return model.RenewNodeCertificateResult{TenantID: in.TenantID, NodeID: in.NodeID, Certificate: bundle.Certificate, Renewed: true}, nil
	}

	dueBefore := time.Now().UTC().Add(s.opts.NodeCertRenewBefore)
	if !in.Force && active.NotAfter.After(dueBefore) {
		return model.RenewNodeCertificateResult{}, &AppError{Status: 409, Code: "certificate_renew_not_due", Message: "node certificate is not due for renewal"}
	}

	bundle, issueErr := s.issueNodeCertificateInternal(ctx, node, &active)
	if issueErr != nil {
		return model.RenewNodeCertificateResult{}, issueErr
	}
	if bundle == nil {
		return model.RenewNodeCertificateResult{}, &AppError{Status: 500, Code: "node_certificate_renew_failed", Message: "certificate renewal returned empty result"}
	}

	s.auditBestEffort(ctx, model.AuditLogEvent{
		TenantID:   node.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "cert_renew",
		TargetType: "node",
		TargetID:   node.ID,
		Metadata: map[string]any{
			"previous_certificate_id": active.ID,
			"previous_serial_number":  active.SerialNumber,
			"certificate_id":          bundle.Certificate.ID,
			"serial_number":           bundle.Certificate.SerialNumber,
		},
		OccurredAt: time.Now().UTC(),
	})

	return model.RenewNodeCertificateResult{
		TenantID:              in.TenantID,
		NodeID:                in.NodeID,
		PreviousCertificateID: active.ID,
		PreviousSerialNumber:  active.SerialNumber,
		Certificate:           bundle.Certificate,
		Renewed:               true,
	}, nil
}

func (s *CoreService) RunPKIRenewalSweep(ctx context.Context, limit int) (int, error) {
	if !s.isPKIStrict() {
		return 0, nil
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return 0, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	expiring, err := cs.ListNodeCertificatesExpiringBefore(ctx, time.Now().UTC().Add(s.opts.NodeCertRenewBefore), limit)
	if err != nil {
		return 0, mapStoreError("certificate_renew_sweep_failed", err)
	}
	renewed := 0
	for _, cert := range expiring {
		node, nodeErr := s.store.GetNodeByID(ctx, cert.TenantID, cert.NodeID)
		if nodeErr != nil {
			continue
		}
		if _, issueErr := s.issueNodeCertificateInternal(ctx, node, &cert); issueErr != nil {
			continue
		}
		renewed++
	}
	return renewed, nil
}
