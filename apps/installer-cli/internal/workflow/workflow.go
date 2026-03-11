package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/compose"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/system"
)

type Service struct {
	cfg     config.Config
	checks  *system.Checker
	compose *compose.Runner
}

func New(cfg config.Config, checks *system.Checker, composeRunner *compose.Runner) *Service {
	return &Service{cfg: cfg, checks: checks, compose: composeRunner}
}

func (s *Service) Install(ctx context.Context) error {
	if err := s.checks.Run(ctx); err != nil {
		return err
	}
	if err := s.ensureBackupDir(); err != nil {
		return err
	}
	if err := s.compose.Config(ctx); err != nil {
		return err
	}
	if err := s.compose.Up(ctx); err != nil {
		return err
	}
	if err := s.checks.CheckGatewayReady(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) Upgrade(ctx context.Context) error {
	if err := s.checks.Run(ctx); err != nil {
		return err
	}
	if err := s.Backup(ctx); err != nil {
		return fmt.Errorf("pre-upgrade backup failed: %w", err)
	}
	if err := s.compose.Pull(ctx); err != nil {
		return err
	}
	if err := s.compose.Up(ctx); err != nil {
		return err
	}
	return s.checks.CheckGatewayReady(ctx)
}

func (s *Service) Rollback(ctx context.Context) error {
	// TODO: Add immutable release snapshot and DB migration rollback policy.
	if err := s.checks.Run(ctx); err != nil {
		return err
	}
	return s.compose.Up(ctx)
}

func (s *Service) Backup(ctx context.Context) error {
	if err := s.ensureBackupDir(); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	marker := filepath.Join(s.cfg.BackupDir, "backup-"+stamp+".meta")
	data := []byte("opener-netdoor backup marker\n")
	if err := os.WriteFile(marker, data, 0o600); err != nil {
		return fmt.Errorf("write backup marker: %w", err)
	}
	_ = ctx
	// TODO: Add consistent PostgreSQL dump + encrypted archive + retention policy.
	return nil
}

func (s *Service) Restore(ctx context.Context) error {
	_ = ctx
	// TODO: Add backup catalog selection and restore transaction boundary.
	return fmt.Errorf("restore workflow seam is defined but not fully implemented")
}

func (s *Service) Doctor(ctx context.Context) error {
	if err := s.checks.Run(ctx); err != nil {
		return err
	}
	if err := s.compose.Config(ctx); err != nil {
		return err
	}
	return s.checks.CheckGatewayReady(ctx)
}

func (s *Service) ensureBackupDir() error {
	return os.MkdirAll(s.cfg.BackupDir, 0o755)
}
