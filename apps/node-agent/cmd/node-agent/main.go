package main

import (
	"log"
	"os"

	"github.com/opener-netdoor/opener-netdoor/apps/node-agent/internal/agent"
)

func main() {
	cfg := agent.Config{
		NodeID:         getenv("NODE_ID", "node-local-1"),
		ControlPlane:   getenv("CONTROL_PLANE_URL", "http://127.0.0.1:8081"),
		HeartbeatEvery: getenv("HEARTBEAT_EVERY", "15s"),
	}
	if err := agent.Run(cfg); err != nil {
		log.Fatalf("node-agent failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}


