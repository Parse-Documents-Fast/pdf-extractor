package config_test

import (
	"testing"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.StreamExtraction != "queue:extraction" {
		t.Errorf("StreamExtraction = %q", cfg.StreamExtraction)
	}
	if cfg.StreamResults != "queue:extraction-results" {
		t.Errorf("StreamResults = %q", cfg.StreamResults)
	}
	if cfg.StreamDLQ != "queue:extraction-dlq" {
		t.Errorf("StreamDLQ = %q", cfg.StreamDLQ)
	}
	if cfg.RedisConsumerGroup != "pdf-extractor-group" {
		t.Errorf("RedisConsumerGroup = %q", cfg.RedisConsumerGroup)
	}
	if cfg.HTTPAddr != ":8000" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.WorkerCount != 1 {
		t.Errorf("WorkerCount = %d, want 1", cfg.WorkerCount)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.PDFMaxBytes != 20971520 {
		t.Errorf("PDFMaxBytes = %d, want 20971520", cfg.PDFMaxBytes)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestLoadCustomValues(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://custom:6379/1")
	t.Setenv("WORKER_COUNT", "4")
	t.Setenv("PDF_MAX_BYTES", "1048576")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisURL != "redis://custom:6379/1" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.WorkerCount != 4 {
		t.Errorf("WorkerCount = %d, want 4", cfg.WorkerCount)
	}
	if cfg.PDFMaxBytes != 1048576 {
		t.Errorf("PDFMaxBytes = %d, want 1048576", cfg.PDFMaxBytes)
	}
}
