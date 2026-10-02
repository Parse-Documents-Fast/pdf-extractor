package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/models"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/repository"
)

func startMongo(t *testing.T) *mongo.Database {
	t.Helper()
	ctx := context.Background()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Skipf("no se pudo levantar MongoDB (testcontainers): %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		t.Fatalf("mongo.Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})

	return client.Database("pdf_extractor_test")
}

func TestMongoPdfRepository(t *testing.T) {
	db := startMongo(t)
	ctx := context.Background()

	t.Run("upsert inserts new document", func(t *testing.T) {
		repo := repository.NewMongoPdfRepository(db, "docs_insert")

		doc := models.PdfDocument{
			PdfID:           "pdf-1",
			Filename:        "a.pdf",
			MarkdownContent: "# Titulo",
			ExtractedAt:     time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
			Status:          models.StatusSuccess,
		}

		if err := repo.Upsert(ctx, doc); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}

		var got models.PdfDocument
		if err := db.Collection("docs_insert").FindOne(ctx, bson.M{"pdf_id": "pdf-1"}).Decode(&got); err != nil {
			t.Fatalf("FindOne() error = %v", err)
		}
		if got.Filename != "a.pdf" {
			t.Errorf("Filename = %q, want a.pdf", got.Filename)
		}
		if got.MarkdownContent != "# Titulo" {
			t.Errorf("MarkdownContent = %q, want # Titulo", got.MarkdownContent)
		}
		if got.Status != models.StatusSuccess {
			t.Errorf("Status = %q, want %s", got.Status, models.StatusSuccess)
		}
	})

	t.Run("upsert is idempotent", func(t *testing.T) {
		repo := repository.NewMongoPdfRepository(db, "docs_idem")

		doc := models.PdfDocument{
			PdfID:           "pdf-2",
			Filename:        "a.pdf",
			ExtractedAt:     time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
			Status:          models.StatusSuccess,
			MarkdownContent: "# V1",
		}
		if err := repo.Upsert(ctx, doc); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}

		doc.Filename = "b.pdf"
		doc.MarkdownContent = "# V2"
		if err := repo.Upsert(ctx, doc); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}

		count, err := db.Collection("docs_idem").CountDocuments(ctx, bson.M{"pdf_id": "pdf-2"})
		if err != nil {
			t.Fatalf("CountDocuments() error = %v", err)
		}
		if count != 1 {
			t.Errorf("CountDocuments() = %d, want 1", count)
		}

		var got models.PdfDocument
		if err := db.Collection("docs_idem").FindOne(ctx, bson.M{"pdf_id": "pdf-2"}).Decode(&got); err != nil {
			t.Fatalf("FindOne() error = %v", err)
		}
		if got.Filename != "b.pdf" {
			t.Errorf("Filename = %q, want b.pdf", got.Filename)
		}
		if got.MarkdownContent != "# V2" {
			t.Errorf("MarkdownContent = %q, want # V2", got.MarkdownContent)
		}
	})

	t.Run("ensure index enforces uniqueness", func(t *testing.T) {
		repo := repository.NewMongoPdfRepository(db, "docs_idx")

		if err := repo.EnsureIndex(ctx); err != nil {
			t.Fatalf("EnsureIndex() error = %v", err)
		}

		col := db.Collection("docs_idx")
		base := models.PdfDocument{
			PdfID:       "dup",
			ExtractedAt: time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
			Status:      models.StatusSuccess,
		}

		if _, err := col.InsertOne(ctx, base); err != nil {
			t.Fatalf("InsertOne() error = %v", err)
		}
		if _, err := col.InsertOne(ctx, base); err == nil {
			t.Fatal("InsertOne() esperaba error de clave duplicada")
		}
	})

	t.Run("upsert with empty pdf id", func(t *testing.T) {
		repo := repository.NewMongoPdfRepository(db, "docs_empty")

		err := repo.Upsert(ctx, models.PdfDocument{})
		if !errors.Is(err, repository.ErrEmptyPdfID) {
			t.Fatalf("Upsert() error = %v, want ErrEmptyPdfID", err)
		}
	})
}
