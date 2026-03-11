package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/compose"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/system"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/workflow"
)

type Runner struct {
	cfg      config.Config
	checks   *system.Checker
	compose  *compose.Runner
	workflow *workflow.Service
}

func New(cfg config.Config) *Runner {
	checker := system.NewChecker(cfg)
	composeRunner := compose.NewRunner(cfg)
	wf := workflow.New(cfg, checker, composeRunner)
	return &Runner{cfg: cfg, checks: checker, compose: composeRunner, workflow: wf}
}

func (r *Runner) Run(ctx context.Context, command string) error {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "install":
		return r.workflow.Install(ctx)
	case "upgrade":
		return r.workflow.Upgrade(ctx)
	case "rollback":
		return r.workflow.Rollback(ctx)
	case "backup":
		return r.workflow.Backup(ctx)
	case "restore":
		return r.workflow.Restore(ctx)
	case "doctor":
		return r.workflow.Doctor(ctx)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}
