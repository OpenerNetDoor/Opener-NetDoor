package config

import (
	"errors"
	"os"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	RedisAddr           string
	NATSURL             string
	NodeSigningSecret   string
	NodeContractVersion string
}

func Load() Config {
	return Config{
		HTTPAddr:            getenv("HTTP_ADDR", ":8081"),
		DatabaseURL:         getenv("DATABASE_URL", "postgresql://openernetdoor:openernetdoor@127.0.0.1:5432/openernetdoor?sslmode=disable"),
		RedisAddr:           getenv("REDIS_ADDR", "127.0.0.1:6379"),
		NATSURL:             getenv("NATS_URL", "nats://127.0.0.1:4222"),
		NodeSigningSecret:   getenv("NODE_SIGNING_SECRET", "opener-netdoor-stage5-dev-signing-secret"),
		NodeContractVersion: getenv("NODE_CONTRACT_VERSION", "2026-03-10.stage5.v1"),
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
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
