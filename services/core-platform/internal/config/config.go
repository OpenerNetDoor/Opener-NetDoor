package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr                string
	DatabaseURL             string
	RedisAddr               string
	NATSURL                 string
	NodeSigningSecret       string
	NodeContractVersion     string
	NodePKIMode             string
	NodeCAMode              string
	NodeCAActiveIssuerID    string
	NodeCAPreviousIssuerIDs []string
	NodeCACertPath          string
	NodeCAKeyPath           string
	NodeCertRenewBefore     time.Duration
	NodeCertDefaultTTL      time.Duration
	NodeCertMaxTTL          time.Duration
	NodeLegacyHMACFallback  bool
}

func Load() Config {
	return Config{
		HTTPAddr:                getenv("HTTP_ADDR", ":8081"),
		DatabaseURL:             getenv("DATABASE_URL", "postgresql://openernetdoor:openernetdoor@127.0.0.1:5432/openernetdoor?sslmode=disable"),
		RedisAddr:               getenv("REDIS_ADDR", "127.0.0.1:6379"),
		NATSURL:                 getenv("NATS_URL", "nats://127.0.0.1:4222"),
		NodeSigningSecret:       getenv("NODE_SIGNING_SECRET", "opener-netdoor-stage5-dev-signing-secret"),
		NodeContractVersion:     getenv("NODE_CONTRACT_VERSION", "2026-03-10.stage5.v1"),
		NodePKIMode:             strings.ToLower(getenv("NODE_PKI_MODE", "strict")),
		NodeCAMode:              strings.ToLower(getenv("NODE_CA_MODE", "file")),
		NodeCAActiveIssuerID:    strings.TrimSpace(getenv("NODE_CA_ACTIVE_ISSUER_ID", "default-file-issuer")),
		NodeCAPreviousIssuerIDs: parseCSV(getenv("NODE_CA_PREVIOUS_ISSUER_IDS", "")),
		NodeCACertPath:          strings.TrimSpace(getenv("NODE_CA_CERT_PATH", "")),
		NodeCAKeyPath:           strings.TrimSpace(getenv("NODE_CA_KEY_PATH", "")),
		NodeCertRenewBefore:     parseDuration(getenv("NODE_CERT_RENEW_BEFORE", "168h"), 168*time.Hour),
		NodeCertDefaultTTL:      parseDuration(getenv("NODE_CERT_DEFAULT_TTL", "720h"), 720*time.Hour),
		NodeCertMaxTTL:          parseDuration(getenv("NODE_CERT_MAX_TTL", "720h"), 720*time.Hour),
		NodeLegacyHMACFallback:  parseBool(getenv("NODE_LEGACY_HMAC_FALLBACK", "false"), false),
	}
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.RedisAddr == "" {
		return errors.New("REDIS_ADDR is required")
	}
	if c.NATSURL == "" {
		return errors.New("NATS_URL is required")
	}
	if len(c.NodeSigningSecret) < 16 {
		return errors.New("NODE_SIGNING_SECRET must be at least 16 characters")
	}
	if c.NodeContractVersion == "" {
		return errors.New("NODE_CONTRACT_VERSION is required")
	}
	if c.NodePKIMode != "strict" && c.NodePKIMode != "legacy" {
		return errors.New("NODE_PKI_MODE must be strict or legacy")
	}
	if c.NodeCAMode != "file" && c.NodeCAMode != "external" {
		return errors.New("NODE_CA_MODE must be file or external")
	}
	if c.NodeCAActiveIssuerID == "" {
		return errors.New("NODE_CA_ACTIVE_ISSUER_ID is required")
	}
	if c.NodeCertRenewBefore <= 0 {
		return errors.New("NODE_CERT_RENEW_BEFORE must be > 0")
	}
	if c.NodeCertDefaultTTL <= 0 {
		return errors.New("NODE_CERT_DEFAULT_TTL must be > 0")
	}
	if c.NodeCertMaxTTL <= 0 {
		return errors.New("NODE_CERT_MAX_TTL must be > 0")
	}
	if c.NodeCertDefaultTTL > c.NodeCertMaxTTL {
		return errors.New("NODE_CERT_DEFAULT_TTL must be <= NODE_CERT_MAX_TTL")
	}
	if c.NodeCAMode == "file" {
		if (c.NodeCACertPath == "") != (c.NodeCAKeyPath == "") {
			return errors.New("NODE_CA_CERT_PATH and NODE_CA_KEY_PATH must be both set or both empty")
		}
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func parseBool(raw string, fallback bool) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
