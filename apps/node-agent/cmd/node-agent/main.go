package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/opener-netdoor/opener-netdoor/apps/node-agent/internal/agent"
	"github.com/opener-netdoor/opener-netdoor/apps/node-agent/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("node-agent config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx, cfg); err != nil {
		log.Fatalf("node-agent failed: %v", err)
	}
}
