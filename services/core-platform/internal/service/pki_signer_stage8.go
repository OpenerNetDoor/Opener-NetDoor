package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/model"
)

type signedNodeLeaf struct {
	SerialHex     string
	CertPEM       string
	PrivateKeyPEM string
	NotBefore     time.Time
	NotAfter      time.Time
}

type nodeCertificateSigner interface {
	Mode() string
	SignNodeLeaf(node model.Node, ttl time.Duration) (signedNodeLeaf, error)
}

type fileNodeCertificateSigner struct {
	bundle *caBundle
}

func (s fileNodeCertificateSigner) Mode() string {
	return "file"
}

func (s fileNodeCertificateSigner) SignNodeLeaf(node model.Node, ttl time.Duration) (signedNodeLeaf, error) {
	if s.bundle == nil || s.bundle.Cert == nil || s.bundle.Key == nil {
		return signedNodeLeaf{}, &AppError{Status: 500, Code: "node_pki_init_failed", Message: "missing CA bundle"}
	}
	if ttl <= 0 {
		return signedNodeLeaf{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "invalid certificate ttl"}
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return signedNodeLeaf{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "failed to generate node certificate key", Err: err}
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 126))
	if err != nil {
		return signedNodeLeaf{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "failed to generate certificate serial", Err: err}
	}
	now := time.Now().UTC()
	notBefore := now.Add(-2 * time.Minute)
	notAfter := now.Add(ttl)
	leaf := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   node.ID,
			Organization: []string{node.TenantID},
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, s.bundle.Cert, &leafKey.PublicKey, s.bundle.Key)
	if err != nil {
		return signedNodeLeaf{}, &AppError{Status: 500, Code: "node_certificate_issue_failed", Message: "failed to create certificate", Err: err}
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	privateKeyDER := x509.MarshalPKCS1PrivateKey(leafKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyDER})

	return signedNodeLeaf{
		SerialHex:     strings.ToUpper(serial.Text(16)),
		CertPEM:       string(certPEM),
		PrivateKeyPEM: string(privateKeyPEM),
		NotBefore:     notBefore,
		NotAfter:      notAfter,
	}, nil
}

type externalNodeCertificateSigner struct{}

func (externalNodeCertificateSigner) Mode() string {
	return "external"
}

func (externalNodeCertificateSigner) SignNodeLeaf(_ model.Node, _ time.Duration) (signedNodeLeaf, error) {
	return signedNodeLeaf{}, &AppError{Status: 501, Code: "external_signer_not_implemented", Message: "external signer mode is not implemented yet"}
}

func (s *CoreService) nodeCertificateSigner() (nodeCertificateSigner, error) {
	switch strings.ToLower(strings.TrimSpace(s.opts.NodeCAMode)) {
	case "external":
		return externalNodeCertificateSigner{}, nil
	case "file", "":
		if err := s.ensureCA(); err != nil {
			return nil, err
		}
		if s.caBundle == nil {
			return nil, &AppError{Status: 500, Code: "node_pki_init_failed", Message: "missing CA bundle"}
		}
		return fileNodeCertificateSigner{bundle: s.caBundle}, nil
	default:
		return nil, &AppError{Status: 400, Code: "validation_error", Message: "unsupported NODE_CA_MODE"}
	}
}

func (s *CoreService) certificateTTL(ttlSeconds *int) time.Duration {
	ttl := s.opts.NodeCertDefaultTTL
	if ttlSeconds != nil && *ttlSeconds > 0 {
		ttl = time.Duration(*ttlSeconds) * time.Second
	}
	if ttl <= 0 {
		ttl = s.opts.NodeCertMaxTTL
	}
	if ttl > s.opts.NodeCertMaxTTL {
		ttl = s.opts.NodeCertMaxTTL
	}
	return ttl
}
