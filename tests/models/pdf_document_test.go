package models_test

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/models"
)

func TestPdfDocumentBSONFieldNames(t *testing.T) {
	doc := models.PdfDocument{
		PdfID:           "pdf-123",
		Filename:        "informe.pdf",
		MarkdownContent: "# Titulo\n\nContenido",
		ExtractedAt:     time.Date(2026, 10, 2, 10, 30, 0, 0, time.UTC),
		Status:          models.StatusSuccess,
	}

	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}

	var raw bson.M
	if err := bson.Unmarshal(data, &raw); err != nil {
		t.Fatalf("bson.Unmarshal() error = %v", err)
	}

	wantKeys := []string{"pdf_id", "filename", "markdown_content", "extracted_at", "status"}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("campo bson %q no encontrado", key)
		}
	}

	if raw["pdf_id"] != "pdf-123" {
		t.Errorf("pdf_id = %v, want pdf-123", raw["pdf_id"])
	}
	if raw["filename"] != "informe.pdf" {
		t.Errorf("filename = %v, want informe.pdf", raw["filename"])
	}
	if raw["markdown_content"] != "# Titulo\n\nContenido" {
		t.Errorf("markdown_content = %v", raw["markdown_content"])
	}
	if raw["status"] != models.StatusSuccess {
		t.Errorf("status = %v, want %s", raw["status"], models.StatusSuccess)
	}

	if _, ok := raw["error"]; ok {
		t.Errorf("error no deberia serializarse cuando esta vacio")
	}
}

func TestPdfDocumentBSONErrorPresent(t *testing.T) {
	doc := models.PdfDocument{
		PdfID:       "pdf-456",
		Filename:    "corrupto.pdf",
		ExtractedAt: time.Date(2026, 10, 2, 10, 30, 0, 0, time.UTC),
		Status:      models.StatusFailed,
		Error:       "PDF corrupto",
	}

	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}

	var raw bson.M
	if err := bson.Unmarshal(data, &raw); err != nil {
		t.Fatalf("bson.Unmarshal() error = %v", err)
	}

	if raw["error"] != "PDF corrupto" {
		t.Errorf("error = %v, want 'PDF corrupto'", raw["error"])
	}
}

func TestPdfDocumentBSONRoundTrip(t *testing.T) {
	original := models.PdfDocument{
		PdfID:           "pdf-789",
		Filename:        "tabla.pdf",
		MarkdownContent: "| a | b |\n|---|---|",
		ExtractedAt:     time.Date(2026, 10, 2, 10, 30, 0, 0, time.UTC),
		Status:          models.StatusSuccess,
		Error:           "algun error",
	}

	data, err := bson.Marshal(original)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}

	var got models.PdfDocument
	if err := bson.Unmarshal(data, &got); err != nil {
		t.Fatalf("bson.Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(original, got) {
		t.Errorf("round-trip = %+v, want %+v", got, original)
	}
}

func TestStatusConstants(t *testing.T) {
	if models.StatusSuccess != "success" {
		t.Errorf("StatusSuccess = %q, want success", models.StatusSuccess)
	}
	if models.StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q, want failed", models.StatusFailed)
	}
}
