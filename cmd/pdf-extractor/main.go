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

	rdb, err := redis.New(ctx, cfg.RedisURL, 5*time.Second)
	if err != nil {
		return err
	}
	defer rdb.Close()

	srv := httpserver.New(cfg)
	go func() {
		slog.Info("iniciando servidor http", "addr", cfg.HTTPAddr)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("servidor http fallo", "error", err)
		}
	}()

	c := consumer.New(rdb, cfg, logger)
	go func() {
		if err := c.Run(ctx); err != nil {
			slog.Error("consumer fallo", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("apagando servicio...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("error apagando servidor http", "error", err)
	}

	slog.Info("servicio detenido limpiamente")
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
