package main

import (
	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/handler"
	"be-lonceng_unman/internal/middleware"
	"be-lonceng_unman/internal/routes"
	"be-lonceng_unman/internal/services"
	"be-lonceng_unman/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		slog.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize logger
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	// Create root context with cancel for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	port := config.GetRunningPort()

	// Initialize services
	pdfService := services.NewPDFService(nil)
	cacheService := services.NewCacheService(ctx)
	fileStorage := storage.NewFileStorage(".")

	// Initialize handlers
	krsHandler := handler.NewKRSHandler(pdfService, cacheService, fileStorage, log)
	authHandler := handler.NewAuthHandler(log)

	// Initialize middleware
	jwtAuth := middleware.NewJWTAuth(config.GetJWTSecret(), log)

	// Set up rate limiting
	rps := config.GetRateLimitRPS()
	burst := config.GetRateLimitBurst()
	rateLimit := middleware.NewRateLimit(rate.Limit(rps), burst, log, config.GetTrustProxy(), ctx)

	// Set up router
	mux := http.NewServeMux()
	routes.SetupRoutes(config.GetConfig(), mux, jwtAuth, rateLimit, krsHandler, authHandler, log)

	// Create server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in goroutine
	go func() {
		log.Info("Server started", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	log.Info("Server shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server shutdown error", slog.String("error", err.Error()))
	}

	log.Info("Server exited")
}
