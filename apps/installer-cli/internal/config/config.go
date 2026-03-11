package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ProjectName        string
	ComposeFile        string
	EnvFile            string
	DatabaseURL        string
	GatewayURL         string
	BackupDir          string
	RequireDocker      bool
	RequireCompose     bool
	ExpectedGatewayURL string
}

func Load(path string) (Config, error) {
	cfg := Config{
		ProjectName:        getenv("COMPOSE_PROJECT_NAME", "openernetdoor"),
		ComposeFile:        getenv("OPENER_NETDOOR_COMPOSE_FILE", "docker-compose.yml"),
		EnvFile:            getenv("OPENER_NETDOOR_ENV_FILE", ".env"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GatewayURL:         getenv("OPENER_NETDOOR_GATEWAY_URL", "http://127.0.0.1:8080"),
		BackupDir:          getenv("OPENER_NETDOOR_BACKUP_DIR", "ops/backups"),
		RequireDocker:      true,
		RequireCompose:     true,
		ExpectedGatewayURL: "http://core-platform:8081",
	}

	if strings.TrimSpace(path) != "" {
		if err := applyEnvFile(path); err != nil {
			return Config{}, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ProjectName) == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.TrimSpace(c.ComposeFile) == "" {
		return fmt.Errorf("compose file is required")
	}
	if strings.TrimSpace(c.EnvFile) == "" {
		return fmt.Errorf("env file is required")
	}
	if _, err := os.Stat(c.ComposeFile); err != nil {
		return fmt.Errorf("compose file check failed: %w", err)
	}
	return nil
}

func applyEnvFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	f, err := os.Open(abs)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		_ = os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("scan config file: %w", err)
	}
	return nil
}

func getenv(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
