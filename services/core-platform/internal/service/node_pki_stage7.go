package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
)

type caBundle struct {
	ID     string
	Issuer string
	Cert   *x509.Certificate
	Key    *rsa.PrivateKey
}

type issuedNodeCertificate struct {
	Certificate   model.NodeCertificate
	PrivateKeyPEM string
}

func identitySerial(identity *model.NodeTLSIdentity) string {
	if identity == nil {
		return ""
	}
	return strings.TrimSpace(identity.SerialNumber)
}

func (s *CoreService) isPKIStrict() bool {
	return strings.EqualFold(strings.TrimSpace(s.opts.NodePKIMode), "strict")
}

func (s *CoreService) nodeCertStore() (store.NodeCertificateStore, error) {
	cs, ok := s.store.(store.NodeCertificateStore)
	if !ok {
		return nil, &AppError{Status: 500, Code: "node_pki_unavailable", Message: "node certificate store is not configured"}
	}
	return cs, nil
}

func (s *CoreService) ensureCA() error {
	if !s.isPKIStrict() {
		return nil
	}
	s.caInitOnce.Do(func() {
		bundle, err := loadOrCreateCA(s.opts.NodeCACertPath, s.opts.NodeCAKeyPath)
		s.caBundle = bundle
		s.caInitErr = err
	})
	if s.caInitErr != nil {
		return &AppError{Status: 500, Code: "node_pki_init_failed", Message: "failed to initialize node PKI", Err: s.caInitErr}
	}
	return nil
}

func loadOrCreateCA(certPath string, keyPath string) (*caBundle, error) {
	certPath = strings.TrimSpace(certPath)
	keyPath = strings.TrimSpace(keyPath)
	if certPath != "" && keyPath != "" {
		certBytes, certErr := os.ReadFile(certPath)
		keyBytes, keyErr := os.ReadFile(keyPath)
		if certErr == nil && keyErr == nil {
			certBlock, _ := pem.Decode(certBytes)
			keyBlock, _ := pem.Decode(keyBytes)
			if certBlock != nil && keyBlock != nil {
				caCert, err := x509.ParseCertificate(certBlock.Bytes)
				if err == nil {
					caKeyAny, parseErr := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
					if parseErr != nil {
						caKeyAny, parseErr = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
					}
					if parseErr == nil {
						if caKey, ok := caKeyAny.(*rsa.PrivateKey); ok {
							return &caBundle{
								ID:     hashString(string(certBytes)),
								Issuer: caCert.Subject.CommonName,
								Cert:   caCert,
								Key:    caKey,
							}, nil
						}
					}
				}
			}
		}
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "opener-netdoor-local-ca",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return &caBundle{
		ID:     hashString(string(caPEM)),
		Issuer: caCert.Subject.CommonName,
		Cert:   caCert,
		Key:    caKey,
	}, nil
}

func (s *CoreService) ensureNodeCertificate(ctx context.Context, node model.Node, hadExistingNode bool) (*issuedNodeCertificate, error) {
	if !s.isPKIStrict() {
		return nil, nil
	}
	if err := s.ensureCA(); err != nil {
		return nil, err
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return nil, err
	}

	active, err := cs.GetActiveNodeCertificate(ctx, node.TenantID, node.ID)
	if err == nil {
		now := time.Now().UTC()
		if active.RevokedAt == nil && now.After(active.NotBefore) && now.Before(active.NotAfter) {
			return &issuedNodeCertificate{Certificate: active}, nil
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, mapStoreError("node_certificate_issue_failed", err)
	}

	return s.issueNodeCertificateInternal(ctx, node, &active)
}

func (s *CoreService) issueNodeCertificateInternal(ctx context.Context, node model.Node, rotateFrom *model.NodeCertificate) (*issuedNodeCertificate, error) {
	if !s.isPKIStrict() {
		return nil, nil
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return nil, err
	}
	signingIssuer, err := s.ensureSigningIssuer(ctx)
	if err != nil {
		return nil, err
	}
	signer, err := s.nodeCertificateSigner()
	if err != nil {
		return nil, err
	}
	signedLeaf, err := signer.SignNodeLeaf(node, s.certificateTTL(nil))
	if err != nil {
		return nil, err
	}

	var rotateFromID *string
	if rotateFrom != nil && strings.TrimSpace(rotateFrom.ID) != "" {
		rotateFromID = &rotateFrom.ID
		if rotateFrom.RevokedAt == nil {
			_, revokeErr := cs.RevokeNodeCertificateByID(ctx, node.TenantID, node.ID, rotateFrom.ID)
			if revokeErr != nil && !errors.Is(revokeErr, sql.ErrNoRows) {
				return nil, mapStoreError("node_certificate_issue_failed", revokeErr)
			}
		}
	}
	issued, issueErr := cs.IssueNodeCertificate(ctx, model.IssueNodeCertificateRequest{
		TenantID:         node.TenantID,
		NodeID:           node.ID,
		SerialNumber:     signedLeaf.SerialHex,
		CertPEM:          signedLeaf.CertPEM,
		CAID:             signingIssuer.CAID,
		IssuerID:         signingIssuer.IssuerID,
		Issuer:           signingIssuer.IssuerName,
		NotBefore:        signedLeaf.NotBefore,
		NotAfter:         signedLeaf.NotAfter,
		RotateFromCertID: rotateFromID,
	})
	if issueErr != nil {
		return nil, mapStoreError("node_certificate_issue_failed", issueErr)
	}

	if err := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
		TenantID:   node.TenantID,
		ActorType:  "system",
		Action:     "node.certificate_issued",
		TargetType: "node",
		TargetID:   node.ID,
		Metadata: map[string]any{
			"certificate_id": issued.ID,
			"serial_number":  issued.SerialNumber,
			"issuer_id":      issued.IssuerID,
		},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return nil, mapStoreError("node_certificate_issue_failed", err)
	}

	return &issuedNodeCertificate{Certificate: issued, PrivateKeyPEM: signedLeaf.PrivateKeyPEM}, nil
}

func (s *CoreService) verifyNodeTLSIdentity(ctx context.Context, node model.Node, identity *model.NodeTLSIdentity, requestType string) error {
	if !s.isPKIStrict() {
		return nil
	}
	if identity == nil || strings.TrimSpace(identity.SerialNumber) == "" {
		if s.opts.NodeLegacyHMACFallback {
			return nil
		}
		return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "tls_identity.serial_number is required in strict PKI mode"}
	}
	if err := s.ensureCA(); err != nil {
		return err
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return err
	}
	certMeta, err := cs.GetNodeCertificateBySerial(ctx, node.TenantID, node.ID, strings.TrimSpace(identity.SerialNumber))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "node certificate not found", Err: err}
		}
		return mapStoreError("node_certificate_verify_failed", err)
	}
	now := time.Now().UTC()
	if certMeta.RevokedAt != nil {
		return &AppError{Status: 401, Code: "node_certificate_revoked", Message: "node certificate is revoked"}
	}
	if now.Before(certMeta.NotBefore.UTC()) || now.After(certMeta.NotAfter.UTC()) {
		return &AppError{Status: 401, Code: "node_certificate_expired", Message: "node certificate is expired or not yet valid"}
	}
	if err := s.validateIssuerTrust(ctx, certMeta); err != nil {
		return err
	}
	if strings.TrimSpace(identity.CertPEM) != "" {
		if strings.TrimSpace(identity.CertPEM) != strings.TrimSpace(certMeta.CertPEM) {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "presented certificate does not match stored certificate"}
		}
		block, _ := pem.Decode([]byte(identity.CertPEM))
		if block == nil {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "presented certificate is not valid PEM"}
		}
		presented, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "presented certificate parse failed", Err: parseErr}
		}
		if strings.ToUpper(presented.SerialNumber.Text(16)) != strings.ToUpper(certMeta.SerialNumber) {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "presented certificate serial mismatch"}
		}
		pool, poolErr := s.trustedRootPool(ctx, certMeta)
		if poolErr != nil {
			return poolErr
		}
		if _, verifyErr := presented.Verify(x509.VerifyOptions{Roots: pool, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); verifyErr != nil {
			return &AppError{Status: 401, Code: "invalid_node_certificate", Message: "presented certificate verification failed", Err: verifyErr}
		}
	}
	return nil
}

func (s *CoreService) ListNodeCertificates(ctx context.Context, actor model.ActorPrincipal, q model.ListNodeCertificatesQuery) ([]model.NodeCertificate, error) {
	if strings.TrimSpace(q.TenantID) == "" || strings.TrimSpace(q.NodeID) == "" {
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(q.TenantID) {
		return nil, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return nil, err
	}
	items, err := cs.ListNodeCertificates(ctx, q)
	if err != nil {
		return nil, mapStoreError("node_certificate_list_failed", err)
	}
	return items, nil
}

func (s *CoreService) IssueNodeCertificate(ctx context.Context, actor model.ActorPrincipal, in model.RotateNodeCertificateRequest) (model.NodeCertificate, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.NodeID) == "" {
		return model.NodeCertificate{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	node, err := s.store.GetNodeByID(ctx, in.TenantID, in.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.NodeCertificate{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "failed to load node", Err: err}
	}
	bundle, issueErr := s.issueNodeCertificateInternal(ctx, node, nil)
	if issueErr != nil {
		return model.NodeCertificate{}, issueErr
	}
	if bundle == nil {
		return model.NodeCertificate{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "certificate issuance returned empty result"}
	}
	if err := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
		TenantID:   node.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "node.certificate_issue_admin",
		TargetType: "node",
		TargetID:   node.ID,
		Metadata:   map[string]any{"certificate_id": bundle.Certificate.ID, "serial_number": bundle.Certificate.SerialNumber},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return model.NodeCertificate{}, mapStoreError("node_certificate_issue_failed", err)
	}
	return bundle.Certificate, nil
}

func (s *CoreService) RotateNodeCertificate(ctx context.Context, actor model.ActorPrincipal, in model.RotateNodeCertificateRequest) (model.NodeCertificate, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.NodeID) == "" {
		return model.NodeCertificate{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	node, err := s.store.GetNodeByID(ctx, in.TenantID, in.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, &AppError{Status: 404, Code: "node_not_found", Message: "node not found", Err: err}
		}
		return model.NodeCertificate{}, &AppError{Status: 500, Code: "node_certificate_rotate_failed", Message: "failed to load node", Err: err}
	}
	cs, csErr := s.nodeCertStore()
	if csErr != nil {
		return model.NodeCertificate{}, csErr
	}
	active, _ := cs.GetActiveNodeCertificate(ctx, in.TenantID, in.NodeID)
	bundle, issueErr := s.issueNodeCertificateInternal(ctx, node, &active)
	if issueErr != nil {
		return model.NodeCertificate{}, issueErr
	}
	if bundle == nil {
		return model.NodeCertificate{}, &AppError{Status: 500, Code: "node_certificate_rotate_failed", Message: "certificate rotation returned empty result"}
	}
	if err := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
		TenantID:   node.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "node.certificate_rotated",
		TargetType: "node",
		TargetID:   node.ID,
		Metadata:   map[string]any{"certificate_id": bundle.Certificate.ID, "serial_number": bundle.Certificate.SerialNumber},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return model.NodeCertificate{}, mapStoreError("node_certificate_rotate_failed", err)
	}
	return bundle.Certificate, nil
}

func (s *CoreService) RevokeNodeCertificate(ctx context.Context, actor model.ActorPrincipal, in model.RevokeNodeCertificateRequest) (model.NodeCertificate, error) {
	if strings.TrimSpace(in.TenantID) == "" || strings.TrimSpace(in.NodeID) == "" {
		return model.NodeCertificate{}, &AppError{Status: 400, Code: "validation_error", Message: "tenant_id and node_id are required"}
	}
	if !actor.CanAccessTenant(in.TenantID) {
		return model.NodeCertificate{}, &AppError{Status: 403, Code: "forbidden", Message: "actor cannot access requested tenant"}
	}
	cs, err := s.nodeCertStore()
	if err != nil {
		return model.NodeCertificate{}, err
	}
	var item model.NodeCertificate
	switch {
	case strings.TrimSpace(in.CertificateID) != "":
		item, err = cs.RevokeNodeCertificateByID(ctx, in.TenantID, in.NodeID, in.CertificateID)
	case strings.TrimSpace(in.SerialNumber) != "":
		item, err = cs.RevokeNodeCertificateBySerial(ctx, in.TenantID, in.NodeID, in.SerialNumber)
	default:
		active, getErr := cs.GetActiveNodeCertificate(ctx, in.TenantID, in.NodeID)
		if getErr != nil {
			if errors.Is(getErr, sql.ErrNoRows) {
				return model.NodeCertificate{}, &AppError{Status: 404, Code: "node_certificate_not_found", Message: "active node certificate not found", Err: getErr}
			}
			return model.NodeCertificate{}, mapStoreError("node_certificate_revoke_failed", getErr)
		}
		item, err = cs.RevokeNodeCertificateByID(ctx, in.TenantID, in.NodeID, active.ID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.NodeCertificate{}, &AppError{Status: 404, Code: "node_certificate_not_found", Message: "node certificate not found", Err: err}
		}
		return model.NodeCertificate{}, mapStoreError("node_certificate_revoke_failed", err)
	}
	if err := s.store.InsertAuditLog(ctx, model.AuditLogEvent{
		TenantID:   in.TenantID,
		ActorType:  "admin",
		ActorSub:   actor.Subject,
		Action:     "node.certificate_revoked",
		TargetType: "node",
		TargetID:   in.NodeID,
		Metadata:   map[string]any{"certificate_id": item.ID, "serial_number": item.SerialNumber, "note": in.RevocationNote},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return model.NodeCertificate{}, mapStoreError("node_certificate_revoke_failed", err)
	}
	return item, nil
}
