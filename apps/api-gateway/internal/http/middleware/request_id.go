package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/opener-netdoor/opener-netdoor/apps/api-gateway/internal/http/requestid"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		r.Header.Set("X-Request-ID", rid)
		w.Header().Set("X-Request-ID", rid)
		ctx := requestid.WithContext(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "rid-fallback"
	}
	return hex.EncodeToString(b)
}
