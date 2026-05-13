package routes

import (
	"log/slog"
	"net/http"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/handler"
	"be-lonceng_unman/internal/middleware"
)

// SetupRoutes sets up all application routes
func SetupRoutes(cfg config.Config, mux *http.ServeMux, jwtAuth *middleware.JWTAuth, rateLimit *middleware.RateLimit, krsHandler *handler.KRSHandler, authHandler *handler.AuthHandler, log *slog.Logger) {
	// Public endpoints
	mux.HandleFunc("/cek", handler.CekHandler)

	// Auth endpoint — generate JWT token (POST + X-API-KEY)
	mux.HandleFunc("/api/auth/generate", authHandler.GenerateToken)

	// Protected endpoints — require JWT authentication
	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(krsHandler.GetKRSByNIS)))

	// Apply middleware chain: request-id → rate-limit
	protectedHandler := middleware.SetMiddleware(protectedMux,
		middleware.RequestIDMiddleware,
		rateLimit.Handle,
	)

	// Apply API key middleware to all /api/ routes
	apiKeyMiddleware := middleware.CheckApiKey(middleware.APIKeyConfig{
		APIKeyValue:   cfg.APIKey,
		ServerVersion: cfg.ServerVersion,
		Log:           log,
	})

	// Mount protected routes under /api/
	mux.Handle("/api/", apiKeyMiddleware(http.StripPrefix("/api", protectedHandler)))
}
