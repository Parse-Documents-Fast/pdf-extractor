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

type extractJob struct {
	ctx       context.Context
	pdfData   []byte
	pageCount int
	opts      markdown.Options
	result    chan extractResult
}

type extractResult struct {
	content string
	err     error
}

type Server struct {
	queue chan extractJob
	srv   *http.Server
}

func New(cfg config.Config) *Server {
	s := &Server{
		queue: make(chan extractJob, cfg.WorkerQueueSize),
	}

	for range cfg.WorkerCount {
		go s.worker()
	}

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
			title := "Invalid PDF"
			if errors.Is(rerr, pdf.ErrInvalidPassword) {
				title = "Password Protected PDF"
			}
			writeProblem(w, http.StatusUnprocessableEntity, title, rerr.Error(), r.URL.Path)
			return
		}

		pageCount := pdfReader.NumPage()

		result := make(chan extractResult, 1)
		job := extractJob{
			ctx:       r.Context(),
			pdfData:   pdfData,
			pageCount: pageCount,
			opts: markdown.Options{
				HeadingRatioH1: cfg.HeadingRatioH1,
				HeadingRatioH2: cfg.HeadingRatioH2,
				HeadingRatioH3: cfg.HeadingRatioH3,
			},
			result: result,
		}

		// Backpressure: cola llena → 503 inmediato, sin bloquear.
		select {
		case s.queue <- job:
		default:
			writeProblem(w, http.StatusServiceUnavailable,
				"Service unavailable",
				"El servidor esta procesando demasiadas solicitudes. Intente nuevamente.",
				r.URL.Path)
			return
		}

		// Esperar resultado respetando cancelación del request.
		select {
		case res := <-result:
			if res.err != nil {
				title := "Extraction failed"
				if errors.Is(res.err, extractor.ErrCorruptPDF) {
					title = "Corrupt PDF"
				} else if errors.Is(res.err, extractor.ErrNoExtractableText) {
					title = "No extractable text"
				} else if errors.Is(res.err, extractor.ErrPasswordProtected) {
					title = "Password Protected PDF"
				}
				writeProblem(w, http.StatusUnprocessableEntity, title, res.err.Error(), r.URL.Path)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":    res.content,
				"page_count": pageCount,
			})
		case <-r.Context().Done():
			// Cliente desconectado; el worker descartará el resultado cuando llegue.
		}
	})

	s.srv = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: cfg.ExtractionTimeout + 5*time.Second,
	}

	return s
}

// worker es una de las N goroutines fijas que pueden ejecutar ExtractStructure.
// El runtime HTTP queda completamente separado del algoritmo de extracción.
func (s *Server) worker() {
	for job := range s.queue {
		blocks, err := extractor.ExtractStructure(job.ctx, job.pdfData)
		if err != nil {
			job.result <- extractResult{err: err}
			continue
		}
		content := markdown.MapStructureToMarkdown(blocks, job.opts)
		job.result <- extractResult{content: content}
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
	close(s.queue)
	return s.srv.Shutdown(ctx)
}