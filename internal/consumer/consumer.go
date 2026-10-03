package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/dto"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/markdown"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/models"
	redisclient "github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/repository"
)

var ErrInvalidMessage = errors.New("consumer: mensaje invalido")

type Consumer struct {
	rdb    *redisclient.Client
	repo   repository.PdfRepository
	cfg    config.Config
	logger *slog.Logger
}

func New(rdb *redisclient.Client, repo repository.PdfRepository, cfg config.Config, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{rdb: rdb, repo: repo, cfg: cfg, logger: logger}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < c.cfg.WorkerConcurrency; i++ {
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

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.cfg.StreamExtraction, c.cfg.ConsumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("consumer: crear consumer group: %w", err)
	}
	return nil
}

func (c *Consumer) workerLoop(ctx context.Context) {
	for {
		msgs, err := c.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    c.cfg.ConsumerGroup,
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
				if err := c.process(ctx, msg); err != nil {
					c.logger.Error("consumer: error transitorio", "id", msg.ID, "error", err)
				}
			}
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg goredis.XMessage) error {
	raw, ok := msg.Values[dto.DataField]
	if !ok {
		c.logger.Warn("consumer: mensaje sin campo data", "id", msg.ID)
		return c.ack(ctx, msg.ID)
	}
	data, ok := raw.(string)
	if !ok {
		c.logger.Warn("consumer: campo data no es string", "id", msg.ID)
		return c.ack(ctx, msg.ID)
	}

	job, err := parseJob(data)
	if err != nil {
		c.logger.Warn("consumer: mensaje invalido", "id", msg.ID, "error", err)
		return c.ack(ctx, msg.ID)
	}

	pdfData, err := decodeContent(job, c.cfg.PDFMaxBytes)
	if err != nil {
		return c.fail(ctx, msg.ID, job.PdfID, err)
	}

	docCtx, cancel := context.WithTimeout(ctx, c.cfg.ExtractionTimeout)
	defer cancel()

	md, err := c.extractAndMap(docCtx, pdfData)
	if err != nil {
		if isPermanent(err) {
			return c.fail(ctx, msg.ID, job.PdfID, err)
		}
		return err
	}

	doc := models.PdfDocument{
		PdfID:           job.PdfID,
		Filename:        job.Filename,
		MarkdownContent: md,
		ExtractedAt:     time.Now().UTC(),
		Status:          models.StatusSuccess,
	}
	if err := c.repo.Upsert(ctx, doc); err != nil {
		return err
	}

	result := dto.ExtractionResult{
		PdfID:           job.PdfID,
		MarkdownContent: md,
		Status:          dto.StatusDone,
	}
	if err := c.publish(ctx, result); err != nil {
		return err
	}

	return c.ack(ctx, msg.ID)
}

func (c *Consumer) fail(ctx context.Context, msgID, pdfID string, cause error) error {
	if pdfID != "" {
		doc := models.PdfDocument{
			PdfID:       pdfID,
			Status:      models.StatusFailed,
			Error:       cause.Error(),
			ExtractedAt: time.Now().UTC(),
		}
		if err := c.repo.Upsert(ctx, doc); err != nil {
			c.logger.Error("consumer: persistir doc fallido", "pdf_id", pdfID, "error", err)
		}
	}

	result := dto.ExtractionResult{
		PdfID:  pdfID,
		Status: dto.StatusFailed,
		Error:  cause.Error(),
	}
	if err := c.publish(ctx, result); err != nil {
		return err
	}
	return c.ack(ctx, msgID)
}

func (c *Consumer) extractAndMap(ctx context.Context, pdfData []byte) (string, error) {
	blocks, err := extractor.ExtractStructure(ctx, pdfData)
	if err != nil {
		return "", err
	}
	opts := markdown.Options{
		HeadingRatioH1: c.cfg.HeadingRatioH1,
		HeadingRatioH2: c.cfg.HeadingRatioH2,
		HeadingRatioH3: c.cfg.HeadingRatioH3,
	}
	return markdown.MapStructureToMarkdown(blocks, opts), nil
}

func (c *Consumer) publish(ctx context.Context, result dto.ExtractionResult) error {
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
	return c.rdb.XAck(ctx, c.cfg.StreamExtraction, c.cfg.ConsumerGroup, id).Err()
}

func parseJob(data string) (dto.ExtractionJob, error) {
	var job dto.ExtractionJob
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return job, fmt.Errorf("%w: json: %v", ErrInvalidMessage, err)
	}
	if job.PdfID == "" {
		return job, fmt.Errorf("%w: pdf_id vacio", ErrInvalidMessage)
	}
	if job.ContentBase64 == "" {
		return job, fmt.Errorf("%w: content_base64 vacio", ErrInvalidMessage)
	}
	return job, nil
}

func decodeContent(job dto.ExtractionJob, maxBytes int64) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(job.ContentBase64)
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
