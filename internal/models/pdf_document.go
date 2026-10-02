package models

import "time"

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

type PdfDocument struct {
	PdfID           string    `bson:"pdf_id"`
	Filename        string    `bson:"filename"`
	MarkdownContent string    `bson:"markdown_content"`
	ExtractedAt     time.Time `bson:"extracted_at"`
	Status          string    `bson:"status"`
	Error           string    `bson:"error,omitempty"`
}
