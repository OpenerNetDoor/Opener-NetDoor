package compose

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
)

type Runner struct {
	cfg config.Config
}

func NewRunner(cfg config.Config) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) Up(ctx context.Context) error {
	return r.run(ctx, "up", "-d", "--build")
}

func (r *Runner) Pull(ctx context.Context) error {
	return r.run(ctx, "pull")
}

func (r *Runner) Down(ctx context.Context) error {
	return r.run(ctx, "down")
}

func (r *Runner) Config(ctx context.Context) error {
	return r.run(ctx, "config")
}

func (r *Runner) run(ctx context.Context, args ...string) error {
	cmdArgs := []string{"compose", "-f", r.cfg.ComposeFile, "--env-file", r.cfg.EnvFile}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %s failed: %w: %s", strings.Join(cmdArgs, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
