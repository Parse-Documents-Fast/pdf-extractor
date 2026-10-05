package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RedisURL           string
	RedisConsumerGroup string
	ConsumerName       string
	StreamExtraction   string
	StreamResults      string
	StreamDLQ          string
	MaxRetries         int
	RetryBackoff       time.Duration
	HTTPAddr           string
	ShutdownTimeout    time.Duration
	ExtractionTimeout  time.Duration
	PDFMaxBytes        int64
	HeadingRatioH1     float64
	HeadingRatioH2     float64
	HeadingRatioH3     float64
	WorkerCount        int
	LogLevel           string
}

func Load() (Config, error) {
	c := Config{
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		RedisConsumerGroup: getEnv("REDIS_CONSUMER_GROUP", "pdf-extractor-group"),
		ConsumerName:       getEnv("CONSUMER_NAME", "pdf-extractor-1"),
		StreamExtraction:   getEnv("REDIS_STREAM_INPUT", "queue:extraction"),
		StreamResults:      getEnv("REDIS_STREAM_OUTPUT", "queue:extraction-results"),
		StreamDLQ:          getEnv("REDIS_STREAM_DLQ", "queue:extraction-dlq"),
		MaxRetries:         getEnvAsInt("REDIS_MAX_RETRIES", 5),
		RetryBackoff:       getEnvAsDuration("REDIS_RETRY_BACKOFF_MS", 1000*time.Millisecond),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8000"),
		ShutdownTimeout:    getEnvAsDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		ExtractionTimeout:  getEnvAsDuration("PDF_EXTRACTION_TIMEOUT", 30*time.Second),
		PDFMaxBytes:        getEnvAsInt64("PDF_MAX_BYTES", 20971520),
		HeadingRatioH1:     getEnvAsFloat("FONT_SIZE_HEADING_THRESHOLD", 1.5),
		HeadingRatioH2:     getEnvAsFloat("FONT_SIZE_SUBHEADING_THRESHOLD", 1.2),
		HeadingRatioH3:     1.1,
		WorkerCount:        getEnvAsInt("WORKER_COUNT", 1),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}

	if c.RedisURL == "" {
		return c, fmt.Errorf("REDIS_URL es obligatorio")
	}
	return c, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if ms, err := strconv.Atoi(v); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return fallback
}
