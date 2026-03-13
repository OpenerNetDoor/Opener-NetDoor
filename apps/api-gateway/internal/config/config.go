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
	CorePlatformBaseURL     string
	JWTIssuer               string
	JWTAudience             string
	JWTSecret               string
	SessionCookieName       string
	SessionTTL              time.Duration
	SessionSecret           string
	SessionSecure           bool
	AdminMagicSecret        string
	SubscriptionAccessSecret string
	OwnerScopeID            string
	OwnerSubject            string
	PublicBaseURL           string
}

func Load() Config {
	return Config{
		HTTPAddr:                getenv("HTTP_ADDR", ":8080"),
		CorePlatformBaseURL:     getenv("CORE_PLATFORM_BASE_URL", "http://127.0.0.1:8081"),
		JWTIssuer:               getenv("JWT_ISSUER", "opener-netdoor"),
		JWTAudience:             getenv("JWT_AUDIENCE", "opener-netdoor-api"),
		JWTSecret:               getenv("JWT_SECRET", "dev-secret-change-me"),
		SessionCookieName:       getenv("SESSION_COOKIE_NAME", "opener_netdoor_session"),
		SessionTTL:              parseDuration(getenv("SESSION_TTL", "168h"), 168*time.Hour),
		SessionSecret:           getenv("SESSION_SECRET", getenv("JWT_SECRET", "dev-secret-change-me")),
		SessionSecure:           parseBool(getenv("SESSION_SECURE", "false"), false),
		AdminMagicSecret:        strings.TrimSpace(getenv("ADMIN_ACCESS_SECRET", "")),
		SubscriptionAccessSecret: strings.TrimSpace(getenv("SUBSCRIPTION_ACCESS_SECRET", "")),
		OwnerScopeID:            strings.TrimSpace(getenv("OWNER_SCOPE_ID", "")),
		OwnerSubject:            strings.TrimSpace(getenv("OWNER_SUBJECT", "owner")),
		PublicBaseURL:           strings.TrimSpace(getenv("PUBLIC_BASE_URL", "")),
	}
}

func (c Config) Validate() error {
	if c.HTTPAddr == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if c.CorePlatformBaseURL == "" {
		return errors.New("CORE_PLATFORM_BASE_URL is required")
	}
	if len(c.JWTSecret) < 16 {
		return errors.New("JWT_SECRET must be at least 16 characters")
	}
	if len(c.SessionSecret) < 16 {
		return errors.New("SESSION_SECRET must be at least 16 characters")
	}
	if strings.TrimSpace(c.SessionCookieName) == "" {
		return errors.New("SESSION_COOKIE_NAME is required")
	}
	if c.SessionTTL < time.Hour {
		return errors.New("SESSION_TTL must be >= 1h")
	}
	if c.AdminMagicSecret == "" {
		return errors.New("ADMIN_ACCESS_SECRET is required")
	}
	if c.SubscriptionAccessSecret == "" {
		return errors.New("SUBSCRIPTION_ACCESS_SECRET is required")
	}
	if c.OwnerScopeID == "" {
		return errors.New("OWNER_SCOPE_ID is required")
	}
	if c.PublicBaseURL == "" {
		return errors.New("PUBLIC_BASE_URL is required")
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
