package commands

import (
	"context"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/app"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
)

func Run(command string) error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	r := app.New(cfg)
	return r.Run(context.Background(), command)
}
