package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/dto"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/markdown"
	redisclient "github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
)

var ErrInvalidMessage = errors.New("consumer: mensaje invalido")

type Consumer struct {
	rdb    *redisclient.Client
	cfg    config.Config
	logger *slog.Logger
}

func New(rdb *redisclient.Client, cfg config.Config, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{rdb: rdb, cfg: cfg, logger: logger}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := EnsureGroup(ctx, c.rdb, c.cfg.StreamExtraction, c.cfg.RedisConsumerGroup); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < c.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.workerLoop(ctx)
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

func (c *Consumer) workerLoop(ctx context.Context) {
	for {
		msgs, err := c.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    c.cfg.RedisConsumerGroup,
			Consumer: c.cfg.ConsumerName,
			Streams:  []string{c.cfg.StreamExtraction, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, goredis.Nil) {
				continue
			}
			c.logger.Error("consumer: xreadgroup", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				c.processWithRetries(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processWithRetries(ctx context.Context, msg goredis.XMessage) {
	raw, ok := msg.Values[dto.DataField]
	if !ok {
		c.logger.Warn("consumer: mensaje sin campo data", "id", msg.ID)
		_ = c.ack(ctx, msg.ID)
		return
	}
	data, ok := raw.(string)
	if !ok {
		c.logger.Warn("consumer: campo data no es string", "id", msg.ID)
		_ = c.ack(ctx, msg.ID)
		return
	}

	job, err := parseJob(data)
	if err != nil {
		c.logger.Warn("consumer: mensaje invalido", "id", msg.ID, "error", err)
		_ = c.ack(ctx, msg.ID)
		return
	}

	attempt := 1
	for {
		err := c.processOnce(ctx, job)
		if err == nil {
			_ = c.ack(ctx, msg.ID)
			return
		}

		if isPermanent(err) || attempt >= c.cfg.MaxRetries {
			c.logger.Error("consumer: fallo permanente o max reintentos alcanzados", "pdf_id", job.PdfID, "attempt", attempt, "error", err)
			if attempt >= c.cfg.MaxRetries && !isPermanent(err) {
				c.sendToDLQ(ctx, job, err, attempt)
			} else {
				c.publishFailure(ctx, job.PdfID, err)
			}
			_ = c.ack(ctx, msg.ID)
			return
		}

		c.logger.Warn("consumer: error transitorio, reintentando", "pdf_id", job.PdfID, "attempt", attempt, "error", err)
		backoff := c.cfg.RetryBackoff * time.Duration(1<<(attempt-1))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		attempt++
	}
}

func (c *Consumer) processOnce(ctx context.Context, job dto.ExtractionRequest) error {
	pdfData, err := decodeContent(job, c.cfg.PDFMaxBytes)
	if err != nil {
		return err
	}

	docCtx, cancel := context.WithTimeout(ctx, c.cfg.ExtractionTimeout)
	defer cancel()

	blocks, err := extractor.ExtractStructure(docCtx, pdfData)
	if err != nil {
		return err
	}

	opts := markdown.Options{
		HeadingRatioH1: c.cfg.HeadingRatioH1,
		HeadingRatioH2: c.cfg.HeadingRatioH2,
		HeadingRatioH3: c.cfg.HeadingRatioH3,
	}
	md := markdown.MapStructureToMarkdown(blocks, opts)

	result := dto.ExtractionResult{
		PdfID:           job.PdfID,
		MarkdownContent: &md,
		Status:          dto.StatusSuccess,
		Error:           nil,
	}
	return c.publishResult(ctx, result)
}

func (c *Consumer) publishFailure(ctx context.Context, pdfID string, cause error) {
	errMsg := cause.Error()
	result := dto.ExtractionResult{
		PdfID:           pdfID,
		MarkdownContent: nil,
		Status:          dto.StatusFailed,
		Error:           &errMsg,
	}
	_ = c.publishResult(ctx, result)
}

func (c *Consumer) sendToDLQ(ctx context.Context, job dto.ExtractionRequest, cause error, attempts int) {
	dlqMsg := dto.DlqMessage{
		PdfID:           job.PdfID,
		OriginalMessage: job,
		Error:           cause.Error(),
		AttemptCount:    attempts,
		LastAttemptAt:   time.Now().UTC(),
	}
	payload, err := json.Marshal(dlqMsg)
	if err != nil {
		c.logger.Error("consumer: serializar dlq", "error", err)
		return
	}
	_ = c.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: c.cfg.StreamDLQ,
		Values: map[string]interface{}{dto.DataField: string(payload)},
	}).Err()

	c.publishFailure(ctx, job.PdfID, cause)
}

func (c *Consumer) publishResult(ctx context.Context, result dto.ExtractionResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("consumer: serializar resultado: %w", err)
	}
	return c.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: c.cfg.StreamResults,
		Values: map[string]interface{}{dto.DataField: string(payload)},
	}).Err()
}

func (c *Consumer) ack(ctx context.Context, id string) error {
	return c.rdb.XAck(ctx, c.cfg.StreamExtraction, c.cfg.RedisConsumerGroup, id).Err()
}

func parseJob(data string) (dto.ExtractionRequest, error) {
	var job dto.ExtractionRequest
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return job, fmt.Errorf("%w: json: %v", ErrInvalidMessage, err)
	}
	if job.PdfID == "" {
		return job, fmt.Errorf("%w: pdf_id vacio", ErrInvalidMessage)
	}
	if job.ContentB64 == "" {
		return job, fmt.Errorf("%w: content_base64 vacio", ErrInvalidMessage)
	}
	return job, nil
}

func decodeContent(job dto.ExtractionRequest, maxBytes int64) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(job.ContentB64)
	if err != nil {
		return nil, fmt.Errorf("decodificar base64: %w", err)
	}
	if maxBytes > 0 && int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("PDF excede el tamano maximo (%d bytes)", maxBytes)
	}
	return raw, nil
}

func isPermanent(err error) bool {
	return errors.Is(err, extractor.ErrCorruptPDF) ||
		errors.Is(err, extractor.ErrNoExtractableText) ||
		errors.Is(err, extractor.ErrPasswordProtected)
}
