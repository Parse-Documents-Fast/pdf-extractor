package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/consumer"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/mongodb"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/repository"
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

	rdb, err := redis.New(ctx, cfg.RedisURI, cfg.RedisConnectTimeout)
	if err != nil {
		return err
	}
	defer rdb.Close()

	mclient, err := mongodb.New(ctx, cfg.MongoURI, cfg.MongoConnectTimeout)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		_ = mclient.Close(shutdownCtx)
	}()

	repo := repository.NewMongoPdfRepository(mclient.Database(cfg.MongoDB), cfg.MongoCollection)
	if err := repo.EnsureIndex(ctx); err != nil {
		return err
	}

	c := consumer.New(rdb, repo, cfg, logger)
	return c.Run(ctx)
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
