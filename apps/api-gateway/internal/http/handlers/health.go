package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/response"
	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/proxy"
)

func Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "opener-netdoor-api-gateway"})
}

func Ready(px *proxy.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := px.Ready(ctx); err != nil {
			response.Error(w, r, http.StatusServiceUnavailable, "core_platform_unavailable", "core platform is not ready")
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "opener-netdoor-api-gateway"})
	}
}
