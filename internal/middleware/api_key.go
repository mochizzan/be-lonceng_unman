package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/pkg/response"
)

// APIKeyConfig holds configuration for API key middleware
type APIKeyConfig struct {
	APIKeyValue   string
	ServerVersion string
	Log           *slog.Logger
}

// CheckApiKey returns middleware that validates X-API-KEY header.
// Menggunakan crypto/subtle.ConstantTimeCompare untuk mencegah timing attack.
func CheckApiKey(cfg APIKeyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			apiKey := r.Header.Get("X-API-KEY")

			if apiKey == "" {
				if cfg.Log != nil {
					cfg.Log.WarnContext(ctx, "API key missing",
						slog.String("remote_addr", r.RemoteAddr),
						slog.String("path", r.URL.Path),
					)
				}
				response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Membutuhkan Akses API-KEY")
				return
			}

			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.APIKeyValue)) != 1 {
				if cfg.Log != nil {
					cfg.Log.WarnContext(ctx, "Invalid API key attempt",
						slog.String("remote_addr", r.RemoteAddr),
						slog.String("path", r.URL.Path),
					)
				}
				response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "API-KEY tidak valid")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
