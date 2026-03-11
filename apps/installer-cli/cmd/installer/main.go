package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/app"
	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "optional installer config file (.env style)")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: installer [--config path] <install|upgrade|rollback|backup|restore|doctor>")
		fmt.Fprintln(flag.CommandLine.Output(), "example: installer install")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := app.New(cfg)
	if err := r.Run(ctx, flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "command failed: %v\n", err)
		os.Exit(1)
	}
}
