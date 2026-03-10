package config

import (
	"errors"
	"os"
)

type Config struct {
	HTTPAddr            string
	CorePlatformBaseURL string
	JWTIssuer           string
	JWTAudience         string
	JWTSecret           string
}

func Load() Config {
	return Config{
		HTTPAddr:            getenv("HTTP_ADDR", ":8080"),
		CorePlatformBaseURL: getenv("CORE_PLATFORM_BASE_URL", "http://127.0.0.1:8081"),
		JWTIssuer:           getenv("JWT_ISSUER", "opener-netdoor"),
		JWTAudience:         getenv("JWT_AUDIENCE", "opener-netdoor-api"),
		JWTSecret:           getenv("JWT_SECRET", "dev-secret-change-me"),
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
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

