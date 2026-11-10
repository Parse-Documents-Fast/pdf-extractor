package dto

import "time"

// DataField is only used by the DLQ stream. The job and result streams use
// flat fields (see pdf-docs/contracts/contracts-extractor.md).
const DataField = "data"

const (
	StatusDone   = "done"
	StatusFailed = "failed"
)

type ExtractionRequest struct {
	PdfID      string `json:"pdf_id"`
	Filename   string `json:"filename"`
	ContentB64 string `json:"content_base64"`
}

type ExtractionResult struct {
	PdfID   string  `json:"pdf_id"`
	Status  string  `json:"status"`
	Content *string `json:"content,omitempty"`
	// Error carries the RFC 9457 problem serialized as JSON.
	Error *string `json:"error,omitempty"`
}

type DlqMessage struct {
	PdfID           string            `json:"pdf_id"`
	OriginalMessage ExtractionRequest `json:"original_message"`
	Error           string            `json:"error"`
	AttemptCount    int               `json:"attempt_count"`
	LastAttemptAt   time.Time         `json:"last_attempt_at"`
}
