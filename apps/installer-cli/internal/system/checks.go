package system

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
)

type Checker struct {
	cfg config.Config
}

func NewChecker(cfg config.Config) *Checker {
	return &Checker{cfg: cfg}
}

func (c *Checker) Run(ctx context.Context) error {
	if c.cfg.RequireDocker {
		if err := checkBinary(ctx, "docker", "--version"); err != nil {
			return err
		}
	}
	if c.cfg.RequireCompose {
		if err := checkBinary(ctx, "docker", "compose", "version"); err != nil {
			return err
		}
	}
	if _, err := os.Stat(c.cfg.EnvFile); err != nil {
		return fmt.Errorf("env file check failed: %w", err)
	}
	return nil
}

func (c *Checker) CheckGatewayReady(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.cfg.GatewayURL, "/")+"/v1/ready", nil)
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gateway readiness request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway readiness returned status %d", resp.StatusCode)
	}
	return nil
}

func checkBinary(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("required command failed: %s %v: %w: %s", binary, args, err, strings.TrimSpace(string(out)))
	}
	return nil
}
