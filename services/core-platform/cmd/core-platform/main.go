package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/config"
	platformhttp "github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/http"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/service"
	"github.com/opener-netdoor/opener-netdoor/services/core-platform/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("opener-netdoor core-platform config validation failed: %v", err)
	}

	s, err := store.NewSQLStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("opener-netdoor core-platform failed to init store: %v", err)
	}
	defer func() {
		if cerr := s.Close(); cerr != nil {
			log.Printf("opener-netdoor core-platform store close error: %v", cerr)
		}
	}()
	if err := s.Ping(context.Background()); err != nil {
		log.Fatalf("opener-netdoor core-platform database ping failed: %v", err)
	}

	coreSvc := service.New(s, service.Options{
		NodeSigningSecret:        cfg.NodeSigningSecret,
		NodeContractVersion:      cfg.NodeContractVersion,
		NodeHeartbeatInterval:    30 * time.Second,
		NodeStaleAfter:           90 * time.Second,
		NodeOfflineAfter:         5 * time.Minute,
		NodePKIMode:              cfg.NodePKIMode,
		NodeCAMode:               cfg.NodeCAMode,
		NodeCAActiveIssuerID:     cfg.NodeCAActiveIssuerID,
		NodeCAPreviousIssuerIDs:  append([]string(nil), cfg.NodeCAPreviousIssuerIDs...),
		NodeCACertPath:           cfg.NodeCACertPath,
		NodeCAKeyPath:            cfg.NodeCAKeyPath,
		NodeCertRenewBefore:      cfg.NodeCertRenewBefore,
		NodeCertDefaultTTL:       cfg.NodeCertDefaultTTL,
		NodeCertMaxTTL:           cfg.NodeCertMaxTTL,
		NodeLegacyHMACFallback:   cfg.NodeLegacyHMACFallback,
		RuntimeEnabled:           cfg.RuntimeEnabled,
		RuntimePublicHost:        cfg.RuntimePublicHost,
		RuntimeVLESSPort:         cfg.RuntimeVLESSPort,
		RuntimeRealityPrivateKey: cfg.RuntimeRealityPrivateKey,
		RuntimeRealityPublicKey:  cfg.RuntimeRealityPublicKey,
		RuntimeRealityShortID:    cfg.RuntimeRealityShortID,
		RuntimeRealityServerName: cfg.RuntimeRealityServerName,
	})

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()
	startRenewalScheduler(runtimeCtx, coreSvc, cfg.NodeCertRenewBefore)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           platformhttp.NewHandlerWithService(coreSvc),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("opener-netdoor core-platform listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("opener-netdoor core-platform failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	runtimeCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("opener-netdoor core-platform graceful shutdown error: %v", err)
		os.Exit(1)
	}
	log.Println("opener-netdoor core-platform stopped")
}

func startRenewalScheduler(ctx context.Context, svc *service.CoreService, renewBefore time.Duration) {
	interval := time.Hour
	if renewBefore > 0 {
		half := renewBefore / 2
		if half >= time.Minute && half < interval {
			interval = half
		}
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				renewed, err := svc.RunPKIRenewalSweep(context.Background(), 100)
				if err != nil {
					log.Printf("opener-netdoor core-platform pki renewal sweep failed: %v", err)
					continue
				}
				if renewed > 0 {
					log.Printf("opener-netdoor core-platform pki renewal sweep renewed %d certificates", renewed)
				}
			}
		}
	}()
}
