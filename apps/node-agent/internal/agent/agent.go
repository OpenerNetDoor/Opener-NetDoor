package agent

import "log"

type Config struct {
	NodeID         string
	ControlPlane   string
	HeartbeatEvery string
}

func Run(cfg Config) error {
	log.Printf("node-agent started node_id=%s control_plane=%s heartbeat=%s", cfg.NodeID, cfg.ControlPlane, cfg.HeartbeatEvery)
	// TODO: register node, request signed config, reconcile local engines, report usage/health.
	select {}
}
