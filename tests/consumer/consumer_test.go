package consumer_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/consumer"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/dto"
	redisclient "github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
)

type harness struct {
	rdb *redisclient.Client
	cfg config.Config
}

func newHarness(t *testing.T) harness {
	t.Helper()
	ctx := context.Background()

	rdb, err := redisclient.New(ctx, startRedis(t), 5*time.Second)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	cfg := testConfig()

	if err := rdb.XGroupCreateMkStream(ctx, cfg.StreamExtraction, cfg.RedisConsumerGroup, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		t.Fatalf("XGroupCreateMkStream: %v", err)
	}

	return harness{rdb: rdb, cfg: cfg}
}

func testConfig() config.Config {
	return config.Config{
		StreamExtraction:   "queue:extraction",
		StreamResults:      "queue:extraction-results",
		StreamDLQ:          "queue:extraction-dlq",
		RedisConsumerGroup: "pdf-extractor-test-group",
		ConsumerName:       "pdf-extractor-test-1",
		WorkerCount:        2,
		MaxRetries:         3,
		RetryBackoff:       10 * time.Millisecond,
		PDFMaxBytes:        1 << 20,
		ExtractionTimeout:  10 * time.Second,
		HeadingRatioH1:     1.8,
		HeadingRatioH2:     1.4,
		HeadingRatioH3:     1.15,
	}
}

func startRedis(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:7")
	if err != nil {
		t.Skipf("no se pudo levantar Redis (testcontainers): %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate redis: %v", err)
		}
	})

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return uri
}

func TestConsumerProcessesJob(t *testing.T) {
	h := newHarness(t)
	c := consumer.New(h.rdb, h.cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	job := dto.ExtractionRequest{
		PdfID:      "pdf-1",
		Filename:   "simple.pdf",
		ContentB64: base64.StdEncoding.EncodeToString(readFixture(t, "simple.pdf")),
	}
	xadd(t, h.rdb, h.cfg.StreamExtraction, job)

	res := waitResult(t, h.rdb, h.cfg.StreamResults, "pdf-1", 15*time.Second)
	if res.Status != dto.StatusSuccess {
		var errStr string
		if res.Error != nil {
			errStr = *res.Error
		}
		t.Fatalf("status = %q, want success (error=%q)", res.Status, errStr)
	}
	if res.MarkdownContent == nil || *res.MarkdownContent == "" {
		t.Errorf("markdown_content vacio")
	}
}

func TestConsumerCorruptPDF(t *testing.T) {
	h := newHarness(t)
	c := consumer.New(h.rdb, h.cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	job := dto.ExtractionRequest{
		PdfID:      "pdf-bad",
		Filename:   "corrupt.pdf",
		ContentB64: base64.StdEncoding.EncodeToString(readFixture(t, "corrupt.pdf")),
	}
	xadd(t, h.rdb, h.cfg.StreamExtraction, job)

	res := waitResult(t, h.rdb, h.cfg.StreamResults, "pdf-bad", 15*time.Second)
	if res.Status != dto.StatusFailed {
		t.Fatalf("status = %q, want failed", res.Status)
	}
	if res.Error == nil || *res.Error == "" {
		t.Errorf("error vacio")
	}
}

func xadd(t *testing.T, rdb *redisclient.Client, stream string, job dto.ExtractionRequest) {
	t.Helper()
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if err := rdb.XAdd(context.Background(), &goredis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{dto.DataField: string(payload)},
	}).Err(); err != nil {
		t.Fatalf("XAdd: %v", err)
	}
}

func waitResult(t *testing.T, rdb *redisclient.Client, stream, pdfID string, timeout time.Duration) dto.ExtractionResult {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs, err := rdb.XRange(ctx, stream, "-", "+").Result()
		if err == nil {
			for _, m := range msgs {
				s, ok := m.Values[dto.DataField].(string)
				if !ok {
					continue
				}
				var r dto.ExtractionResult
				if json.Unmarshal([]byte(s), &r) == nil && r.PdfID == pdfID {
					return r
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout esperando resultado de %q", pdfID)
	return dto.ExtractionResult{}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
