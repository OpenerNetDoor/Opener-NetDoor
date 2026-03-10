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

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/config"
	gatewayhttp "github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("opener-netdoor api-gateway config validation failed: %v", err)
	}

	handler, err := gatewayhttp.NewHandler(cfg)
	if err != nil {
		log.Fatalf("opener-netdoor api-gateway handler init failed: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("opener-netdoor api-gateway listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("opener-netdoor api-gateway failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("opener-netdoor api-gateway graceful shutdown error: %v", err)
		os.Exit(1)
	}
	log.Println("opener-netdoor api-gateway stopped")
}

