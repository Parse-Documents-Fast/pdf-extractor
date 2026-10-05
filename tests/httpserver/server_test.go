package httpserver_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/dto"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/httpserver"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestExtractEndpointSuccess(t *testing.T) {
	pdfBytes := readFixture(t, "simple.pdf")
	b64 := base64.StdEncoding.EncodeToString(pdfBytes)

	reqBody := dto.ExtractionRequest{
		PdfID:      "test-1",
		Filename:   "simple.pdf",
		ContentB64: b64,
	}
	body, _ := json.Marshal(reqBody)

	cfg, _ := config.Load()
	srv := httpserver.New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/extract", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp["content"] == nil {
		t.Errorf("response content is nil")
	}
	if resp["page_count"] == nil {
		t.Errorf("response page_count is nil")
	}
}

func TestExtractEndpointInvalidBase64(t *testing.T) {
	reqBody := dto.ExtractionRequest{
		PdfID:      "test-2",
		Filename:   "invalid.pdf",
		ContentB64: "not-valid-base64!!!",
	}
	body, _ := json.Marshal(reqBody)

	cfg, _ := config.Load()
	srv := httpserver.New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/extract", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg, _ := config.Load()
	srv := httpserver.New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
}
