package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/consumer"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/httpserver"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	rawLogLevel := os.Getenv("LOG_LEVEL")
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	if rawLogLevel != "" && rawLogLevel != "debug" && rawLogLevel != "info" && rawLogLevel != "warn" && rawLogLevel != "error" {
		logger.Warn("invalid log level provided, defaulting to info", "provided", rawLogLevel)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rdb, err := redis.New(ctx, cfg.RedisURL, 5*time.Second)
	if err != nil {
		return err
	}
	defer rdb.Close()

	srv := httpserver.New(cfg, logger)
	go func() {
		logger.Info("starting http server", "addr", cfg.HTTPAddr)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
		}
	}()

	c := consumer.New(rdb, cfg, logger)
	go func() {
		if err := c.Run(ctx); err != nil {
			logger.Error("consumer failed", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutting down http server", "error", err)
	}

	logger.Info("service stopped cleanly")
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
