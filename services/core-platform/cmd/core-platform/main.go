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

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: platformhttp.NewHandler(s, platformhttp.Options{ServiceOptions: service.Options{
			NodeSigningSecret:     cfg.NodeSigningSecret,
			NodeContractVersion:   cfg.NodeContractVersion,
			NodeHeartbeatInterval: 30 * time.Second,
			NodeStaleAfter:        90 * time.Second,
			NodeOfflineAfter:      5 * time.Minute,
		}}),
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("opener-netdoor core-platform graceful shutdown error: %v", err)
		os.Exit(1)
	}
	log.Println("opener-netdoor core-platform stopped")
}
