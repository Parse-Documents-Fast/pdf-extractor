package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultRedisConnectTimeout = 5 * time.Second
	defaultMongoConnectTimeout = 10 * time.Second
	defaultShutdownTimeout     = 10 * time.Second
)

type Config struct {
	RedisURI            string
	RedisConnectTimeout time.Duration

	MongoURI            string
	MongoDB             string
	MongoCollection     string
	MongoConnectTimeout time.Duration

	StreamExtraction string
	StreamResults    string
	StreamDLQ        string
	ConsumerGroup    string
	ConsumerName     string

	HTTPAddr string

	WorkerConcurrency int
	MaxRetries        int
	PDFMaxBytes       int64
	ExtractionTimeout time.Duration

	HeadingRatioH1 float64
	HeadingRatioH2 float64
	HeadingRatioH3 float64

	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	var (
		cfg Config
		err error
	)

	cfg.RedisURI = os.Getenv("REDIS_URI")
	cfg.MongoURI = os.Getenv("MONGO_URI")
	cfg.MongoDB = os.Getenv("MONGO_DB")
	cfg.MongoCollection = os.Getenv("MONGO_COLLECTION")

	cfg.RedisConnectTimeout, err = durationEnv("REDIS_CONNECT_TIMEOUT", defaultRedisConnectTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.MongoConnectTimeout, err = durationEnv("MONGO_CONNECT_TIMEOUT", defaultMongoConnectTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.StreamExtraction = stringEnv("STREAM_EXTRACTION", "queue:extraction")
	cfg.StreamResults = stringEnv("STREAM_RESULTS", "queue:extraction-results")
	cfg.StreamDLQ = stringEnv("STREAM_DLQ", "queue:extraction-dlq")
	cfg.ConsumerGroup = stringEnv("CONSUMER_GROUP", "pdf-extractor-group")
	cfg.ConsumerName = stringEnv("CONSUMER_NAME", "pdf-extractor-1")

	cfg.HTTPAddr = stringEnv("HTTP_ADDR", ":8080")

	cfg.WorkerConcurrency, err = intEnv("WORKER_CONCURRENCY", 2)
	if err != nil {
		return Config{}, err
	}
	cfg.MaxRetries, err = intEnv("MAX_RETRIES", 3)
	if err != nil {
		return Config{}, err
	}
	cfg.PDFMaxBytes, err = int64Env("PDF_MAX_BYTES", 20971520)
	if err != nil {
		return Config{}, err
	}
	cfg.ExtractionTimeout, err = durationEnv("EXTRACTION_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg.HeadingRatioH1, err = floatEnv("HEADING_RATIO_H1", 1.8)
	if err != nil {
		return Config{}, err
	}
	cfg.HeadingRatioH2, err = floatEnv("HEADING_RATIO_H2", 1.4)
	if err != nil {
		return Config{}, err
	}
	cfg.HeadingRatioH3, err = floatEnv("HEADING_RATIO_H3", 1.15)
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel = stringEnv("LOG_LEVEL", "info")

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"REDIS_URI", c.RedisURI},
		{"MONGO_URI", c.MongoURI},
		{"MONGO_DB", c.MongoDB},
		{"MONGO_COLLECTION", c.MongoCollection},
	}
	for _, r := range required {
		if r.value == "" {
			return fmt.Errorf("config: %s es obligatoria", r.name)
		}
	}

	if c.RedisConnectTimeout <= 0 {
		return fmt.Errorf("config: REDIS_CONNECT_TIMEOUT debe ser mayor a 0")
	}
	if c.MongoConnectTimeout <= 0 {
		return fmt.Errorf("config: MONGO_CONNECT_TIMEOUT debe ser mayor a 0")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("config: SHUTDOWN_TIMEOUT debe ser mayor a 0")
	}
	if c.WorkerConcurrency < 1 {
		return fmt.Errorf("config: WORKER_CONCURRENCY debe ser mayor o igual a 1")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("config: MAX_RETRIES no puede ser negativo")
	}
	if c.PDFMaxBytes <= 0 {
		return fmt.Errorf("config: PDF_MAX_BYTES debe ser mayor a 0")
	}
	if c.ExtractionTimeout <= 0 {
		return fmt.Errorf("config: EXTRACTION_TIMEOUT debe ser mayor a 0")
	}
	if c.HeadingRatioH1 <= 0 {
		return fmt.Errorf("config: HEADING_RATIO_H1 debe ser mayor a 0")
	}
	if c.HeadingRatioH2 <= 0 {
		return fmt.Errorf("config: HEADING_RATIO_H2 debe ser mayor a 0")
	}
	if c.HeadingRatioH3 <= 0 {
		return fmt.Errorf("config: HEADING_RATIO_H3 debe ser mayor a 0")
	}

	return nil
}

func stringEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func intEnv(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s debe ser un entero: %w", key, err)
	}
	return n, nil
}

func int64Env(key string, def int64) (int64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s debe ser un entero: %w", key, err)
	}
	return n, nil
}

func floatEnv(key string, def float64) (float64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s debe ser un numero: %w", key, err)
	}
	return f, nil
}

func durationEnv(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s debe ser una duracion valida: %w", key, err)
	}
	return d, nil
}
