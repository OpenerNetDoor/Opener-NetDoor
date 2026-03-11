package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	NodeID            string
	TenantID          string
	NodeKeyID         string
	NodePublicKey     string
	CorePlatformURL   string
	ContractVersion   string
	AgentVersion      string
	HeartbeatEvery    time.Duration
	RegistrationNonce string
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		NodeID:            getenv("NODE_ID", "node-local-1"),
		TenantID:          getenv("NODE_TENANT_ID", "00000000-0000-0000-0000-000000000001"),
		NodeKeyID:         getenv("NODE_KEY_ID", "node-key-local-1"),
		NodePublicKey:     getenv("NODE_PUBLIC_KEY", "node-public-key-local"),
		CorePlatformURL:   getenv("CONTROL_PLANE_URL", "http://127.0.0.1:8081"),
		ContractVersion:   getenv("NODE_CONTRACT_VERSION", "2026-03-10.stage5.v1"),
		AgentVersion:      getenv("NODE_AGENT_VERSION", "0.2.0"),
		RegistrationNonce: getenv("NODE_REGISTRATION_NONCE", "bootstrap-nonce-0001"),
	}

	heartbeatSec, err := strconv.Atoi(getenv("HEARTBEAT_EVERY_SECONDS", "15"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid HEARTBEAT_EVERY_SECONDS: %w", err)
	}
	cfg.HeartbeatEvery = time.Duration(heartbeatSec) * time.Second

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.NodeID) == "" {
		return fmt.Errorf("NODE_ID is required")
	}
	if strings.TrimSpace(c.TenantID) == "" {
		return fmt.Errorf("NODE_TENANT_ID is required")
	}
	if strings.TrimSpace(c.NodeKeyID) == "" {
		return fmt.Errorf("NODE_KEY_ID is required")
	}
	if strings.TrimSpace(c.NodePublicKey) == "" {
		return fmt.Errorf("NODE_PUBLIC_KEY is required")
	}
	if strings.TrimSpace(c.CorePlatformURL) == "" {
		return fmt.Errorf("CONTROL_PLANE_URL is required")
	}
	if c.HeartbeatEvery < 5*time.Second {
		return fmt.Errorf("HEARTBEAT_EVERY_SECONDS must be >= 5")
	}
	return nil
}

func getenv(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
