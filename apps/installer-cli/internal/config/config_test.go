package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services:{}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	oldCompose := os.Getenv("OPENER_NETDOOR_COMPOSE_FILE")
	oldEnv := os.Getenv("OPENER_NETDOOR_ENV_FILE")
	defer func() {
		_ = os.Setenv("OPENER_NETDOOR_COMPOSE_FILE", oldCompose)
		_ = os.Setenv("OPENER_NETDOOR_ENV_FILE", oldEnv)
	}()

	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("COMPOSE_PROJECT_NAME=openernetdoor\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	_ = os.Setenv("OPENER_NETDOOR_COMPOSE_FILE", composePath)
	_ = os.Setenv("OPENER_NETDOOR_ENV_FILE", envPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ComposeFile != composePath {
		t.Fatalf("unexpected compose file %q", cfg.ComposeFile)
	}
	if cfg.EnvFile != envPath {
		t.Fatalf("unexpected env file %q", cfg.EnvFile)
	}
}
