package httpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/config"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/dto"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/markdown"
)

type Server struct {
	srv *http.Server
}

func New(cfg config.Config) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/extract", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeProblem(w, http.StatusMethodNotAllowed, "Method not allowed", "Only POST is supported", r.URL.Path)
			return
		}

		var req dto.ExtractionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid request", "JSON body invalido: "+err.Error(), r.URL.Path)
			return
		}

		if req.ContentB64 == "" {
			writeProblem(w, http.StatusBadRequest, "Invalid request", "content_base64 es obligatorio", r.URL.Path)
			return
		}

		pdfData, err := base64.StdEncoding.DecodeString(req.ContentB64)
		if err != nil {
			writeProblem(w, http.StatusBadRequest, "Invalid base64", "Error decodificando base64: "+err.Error(), r.URL.Path)
			return
		}

		if cfg.PDFMaxBytes > 0 && int64(len(pdfData)) > cfg.PDFMaxBytes {
			writeProblem(w, http.StatusRequestEntityTooLarge, "Payload too large", "El PDF excede el tamano maximo permitido", r.URL.Path)
			return
		}

		pdfReader, rerr := pdf.NewReader(bytes.NewReader(pdfData), int64(len(pdfData)))
		if rerr != nil {
			status := http.StatusUnprocessableEntity
			title := "Invalid PDF"
			if errors.Is(rerr, pdf.ErrInvalidPassword) {
				title = "Password Protected PDF"
			}
			writeProblem(w, status, title, rerr.Error(), r.URL.Path)
			return
		}

		pageCount := pdfReader.NumPage()

		blocks, err := extractor.ExtractStructure(r.Context(), pdfData)
		if err != nil {
			status := http.StatusUnprocessableEntity
			title := "Extraction failed"
			if errors.Is(err, extractor.ErrCorruptPDF) {
				title = "Corrupt PDF"
			} else if errors.Is(err, extractor.ErrNoExtractableText) {
				title = "No extractable text"
			} else if errors.Is(err, extractor.ErrPasswordProtected) {
				title = "Password Protected PDF"
			}
			writeProblem(w, status, title, err.Error(), r.URL.Path)
			return
		}

		opts := markdown.Options{
			HeadingRatioH1: cfg.HeadingRatioH1,
			HeadingRatioH2: cfg.HeadingRatioH2,
			HeadingRatioH3: cfg.HeadingRatioH3,
		}
		md := markdown.MapStructureToMarkdown(blocks, opts)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content":    md,
			"page_count": pageCount,
		})
	})

	return &Server{
		srv: &http.Server{
			Addr:         cfg.HTTPAddr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}

func writeProblem(w http.ResponseWriter, status int, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     "about:blank",
		"title":    title,
		"status":   status,
		"detail":   detail,
		"instance": instance,
	})
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Handler() http.Handler {
	return s.srv.Handler
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
