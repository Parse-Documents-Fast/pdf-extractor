package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("REDIS_URI", "redis://localhost:6379/0")
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MONGO_DB", "pdf_extractor")
	t.Setenv("MONGO_COLLECTION", "pdf_documents")
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisURI != "redis://localhost:6379/0" {
		t.Errorf("RedisURI = %q", cfg.RedisURI)
	}
	if cfg.RedisConnectTimeout != 5*time.Second {
		t.Errorf("RedisConnectTimeout = %v, want 5s", cfg.RedisConnectTimeout)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI = %q", cfg.MongoURI)
	}
	if cfg.MongoDB != "pdf_extractor" {
		t.Errorf("MongoDB = %q", cfg.MongoDB)
	}
	if cfg.MongoCollection != "pdf_documents" {
		t.Errorf("MongoCollection = %q", cfg.MongoCollection)
	}
	if cfg.MongoConnectTimeout != 10*time.Second {
		t.Errorf("MongoConnectTimeout = %v, want 10s", cfg.MongoConnectTimeout)
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
	if cfg.ConsumerGroup != "pdf-extractor-group" {
		t.Errorf("ConsumerGroup = %q", cfg.ConsumerGroup)
	}
	if cfg.ConsumerName != "pdf-extractor-1" {
		t.Errorf("ConsumerName = %q", cfg.ConsumerName)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.WorkerConcurrency != 2 {
		t.Errorf("WorkerConcurrency = %d, want 2", cfg.WorkerConcurrency)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.PDFMaxBytes != 20971520 {
		t.Errorf("PDFMaxBytes = %d, want 20971520", cfg.PDFMaxBytes)
	}
	if cfg.ExtractionTimeout != 30*time.Second {
		t.Errorf("ExtractionTimeout = %v, want 30s", cfg.ExtractionTimeout)
	}
	if cfg.HeadingRatioH1 != 1.8 {
		t.Errorf("HeadingRatioH1 = %v, want 1.8", cfg.HeadingRatioH1)
	}
	if cfg.HeadingRatioH2 != 1.4 {
		t.Errorf("HeadingRatioH2 = %v, want 1.4", cfg.HeadingRatioH2)
	}
	if cfg.HeadingRatioH3 != 1.15 {
		t.Errorf("HeadingRatioH3 = %v, want 1.15", cfg.HeadingRatioH3)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	tests := []string{"REDIS_URI", "MONGO_URI", "MONGO_DB", "MONGO_COLLECTION"}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			setRequired(t)
			t.Setenv(key, "")

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() expected error for missing %s", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("Load() error = %q, want mention of %s", err, key)
			}
		})
	}
}

func TestLoadCustomValues(t *testing.T) {
	setRequired(t)
	t.Setenv("WORKER_CONCURRENCY", "7")
	t.Setenv("PDF_MAX_BYTES", "1048576")
	t.Setenv("EXTRACTION_TIMEOUT", "45s")
	t.Setenv("HEADING_RATIO_H1", "2.0")
	t.Setenv("REDIS_CONNECT_TIMEOUT", "2s")
	t.Setenv("MONGO_CONNECT_TIMEOUT", "3s")
	t.Setenv("SHUTDOWN_TIMEOUT", "4s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkerConcurrency != 7 {
		t.Errorf("WorkerConcurrency = %d, want 7", cfg.WorkerConcurrency)
	}
	if cfg.PDFMaxBytes != 1048576 {
		t.Errorf("PDFMaxBytes = %d, want 1048576", cfg.PDFMaxBytes)
	}
	if cfg.ExtractionTimeout != 45*time.Second {
		t.Errorf("ExtractionTimeout = %v, want 45s", cfg.ExtractionTimeout)
	}
	if cfg.HeadingRatioH1 != 2.0 {
		t.Errorf("HeadingRatioH1 = %v, want 2.0", cfg.HeadingRatioH1)
	}
	if cfg.RedisConnectTimeout != 2*time.Second {
		t.Errorf("RedisConnectTimeout = %v, want 2s", cfg.RedisConnectTimeout)
	}
	if cfg.MongoConnectTimeout != 3*time.Second {
		t.Errorf("MongoConnectTimeout = %v, want 3s", cfg.MongoConnectTimeout)
	}
	if cfg.ShutdownTimeout != 4*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 4s", cfg.ShutdownTimeout)
	}
}

func TestLoadInvalidInt(t *testing.T) {
	setRequired(t)
	t.Setenv("WORKER_CONCURRENCY", "abc")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid int")
	}
}

func TestLoadInvalidInt64(t *testing.T) {
	setRequired(t)
	t.Setenv("PDF_MAX_BYTES", "abc")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid int64")
	}
}

func TestLoadInvalidFloat(t *testing.T) {
	setRequired(t)
	t.Setenv("HEADING_RATIO_H2", "abc")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid float")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	setRequired(t)
	t.Setenv("EXTRACTION_TIMEOUT", "abc")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid duration")
	}
}

func TestValidateOutOfRange(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantKey string
	}{
		{
			name:    "worker concurrency zero",
			mutate:  func(c *config.Config) { c.WorkerConcurrency = 0 },
			wantKey: "WORKER_CONCURRENCY",
		},
		{
			name:    "max retries negative",
			mutate:  func(c *config.Config) { c.MaxRetries = -1 },
			wantKey: "MAX_RETRIES",
		},
		{
			name:    "pdf max bytes zero",
			mutate:  func(c *config.Config) { c.PDFMaxBytes = 0 },
			wantKey: "PDF_MAX_BYTES",
		},
		{
			name:    "extraction timeout zero",
			mutate:  func(c *config.Config) { c.ExtractionTimeout = 0 },
			wantKey: "EXTRACTION_TIMEOUT",
		},
		{
			name:    "heading ratio h1 zero",
			mutate:  func(c *config.Config) { c.HeadingRatioH1 = 0 },
			wantKey: "HEADING_RATIO_H1",
		},
		{
			name:    "redis connect timeout zero",
			mutate:  func(c *config.Config) { c.RedisConnectTimeout = 0 },
			wantKey: "REDIS_CONNECT_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() expected error")
			}
			if !strings.Contains(err.Error(), tt.wantKey) {
				t.Errorf("Validate() error = %q, want mention of %s", err, tt.wantKey)
			}
		})
	}
}

func TestValidateValid(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func validConfig() config.Config {
	return config.Config{
		RedisURI:            "redis://localhost:6379/0",
		RedisConnectTimeout: 5 * time.Second,
		MongoURI:            "mongodb://localhost:27017",
		MongoDB:             "pdf_extractor",
		MongoCollection:     "pdf_documents",
		MongoConnectTimeout: 10 * time.Second,
		WorkerConcurrency:   2,
		MaxRetries:          3,
		PDFMaxBytes:         20971520,
		ExtractionTimeout:   30 * time.Second,
		HeadingRatioH1:      1.8,
		HeadingRatioH2:      1.4,
		HeadingRatioH3:      1.15,
		ShutdownTimeout:     10 * time.Second,
	}
}
