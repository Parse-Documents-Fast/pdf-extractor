package dto

const (
	DataField = "data"

	StatusDone   = "done"
	StatusFailed = "failed"
)

type ExtractionJob struct {
	PdfID         string `json:"pdf_id"`
	Filename      string `json:"filename"`
	ContentBase64 string `json:"content_base64"`
	Checksum      string `json:"checksum,omitempty"`
}

type ExtractionResult struct {
	PdfID           string `json:"pdf_id"`
	MarkdownContent string `json:"markdown_content"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}
