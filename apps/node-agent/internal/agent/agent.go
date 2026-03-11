package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/node-agent/internal/config"
)

func Run(ctx context.Context, cfg config.Config) error {
	c := newClient(cfg)
	log.Printf("node-agent starting node_id=%s tenant_id=%s control_plane=%s", cfg.NodeID, cfg.TenantID, cfg.CorePlatformURL)

	nodeID, err := c.register(ctx, cfg)
	if err != nil {
		return fmt.Errorf("register node: %w", err)
	}
	log.Printf("node-agent registered node_id=%s", nodeID)

	ticker := time.NewTicker(cfg.HeartbeatEvery)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("node-agent stopping: %v", ctx.Err())
			return nil
		case <-ticker.C:
			counter++
			nonce := fmt.Sprintf("heartbeat-%d", counter)
			if err := c.heartbeat(ctx, cfg, nodeID, nonce); err != nil {
				log.Printf("heartbeat failed: %v", err)
				continue
			}
			log.Printf("heartbeat accepted node_id=%s nonce=%s", nodeID, nonce)
		}
	}
}
