package dto

import "time"

const DataField = "data"

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

type ExtractionRequest struct {
	PdfID      string `json:"pdf_id"`
	Filename   string `json:"filename"`
	ContentB64 string `json:"content_base64"`
}

type ExtractionResult struct {
	PdfID           string  `json:"pdf_id"`
	MarkdownContent *string `json:"markdown_content"`
	Status          string  `json:"status"`
	Error           *string `json:"error"`
}

type DlqMessage struct {
	PdfID           string            `json:"pdf_id"`
	OriginalMessage ExtractionRequest `json:"original_message"`
	Error           string            `json:"error"`
	AttemptCount    int               `json:"attempt_count"`
	LastAttemptAt   time.Time         `json:"last_attempt_at"`
}
