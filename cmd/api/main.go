package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/views"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize metrics collector
	metrics := views.NewMetricsCollector()
	handlers := views.NewHandlers(metrics)

	// Setup routes
	mux := http.NewServeMux()

	// Health and metrics endpoints
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/metrics", handlers.MetricsHandler)

	// Apply metrics middleware
	handler := handlers.MetricsMiddleware(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server error channel
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		slog.Info("starting HTTP server", "addr", cfg.HTTPAddr)
		errChan <- server.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		return err
	}

	slog.Info("server shutdown complete")
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}