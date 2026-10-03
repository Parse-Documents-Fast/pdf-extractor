package extractor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/ledongthuc/pdf"
)

type Block struct {
	Text     string
	FontSize float64
	X        float64
	Y        float64
	Width    float64
	Page     int
}

type PdfExtractionError struct {
	Op  string
	Err error
}

func (e *PdfExtractionError) Error() string {
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("extractor: %s: %v", e.Op, e.Err)
}

func (e *PdfExtractionError) Unwrap() error {
	return e.Err
}

var (
	ErrCorruptPDF        = errors.New("extractor: PDF corrupto o no legible")
	ErrNoExtractableText = errors.New("extractor: sin texto extraible")
	ErrPasswordProtected = errors.New("extractor: PDF protegido con contraseña")
)

func ExtractStructure(ctx context.Context, pdfData []byte) ([]Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(pdfData) == 0 {
		return nil, &PdfExtractionError{Op: "ExtractStructure", Err: ErrCorruptPDF}
	}

	blocks, err := extract(ctx, pdfData)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, &PdfExtractionError{Op: "ExtractStructure", Err: ErrNoExtractableText}
	}

	return blocks, nil
}

func extract(ctx context.Context, pdfData []byte) (blocks []Block, err error) {
	defer func() {
		if r := recover(); r != nil {
			blocks = nil
			err = &PdfExtractionError{Op: "ExtractStructure", Err: fmt.Errorf("%w: panic: %v", ErrCorruptPDF, r)}
		}
	}()

	r, rerr := pdf.NewReader(bytes.NewReader(pdfData), int64(len(pdfData)))
	if rerr != nil {
		return nil, classifyReaderError(rerr)
	}

	for p := 1; p <= r.NumPage(); p++ {
		if cerr := ctx.Err(); cerr != nil {
			return nil, cerr
		}
		blocks = append(blocks, pageBlocks(r.Page(p), p)...)
	}

	return blocks, nil
}

func classifyReaderError(err error) error {
	if errors.Is(err, pdf.ErrInvalidPassword) {
		return &PdfExtractionError{Op: "NewReader", Err: ErrPasswordProtected}
	}
	return &PdfExtractionError{Op: "NewReader", Err: fmt.Errorf("%w: %v", ErrCorruptPDF, err)}
}

func pageBlocks(page pdf.Page, pageNum int) []Block {
	texts := page.Content().Text
	if len(texts) == 0 {
		return nil
	}

	sort.SliceStable(texts, func(i, j int) bool {
		if texts[i].Y != texts[j].Y {
			return texts[i].Y > texts[j].Y
		}
		return texts[i].X < texts[j].X
	})

	var blocks []Block
	var cur *Block
	var curRight float64

	flush := func() {
		if cur != nil {
			cur.Width = estimateWidth(cur.FontSize, cur.Text)
			blocks = append(blocks, *cur)
		}
	}

	for _, t := range texts {
		if cur == nil {
			cur = newBlock(t, pageNum)
			curRight = t.X
			continue
		}

		sameSize := t.FontSize == cur.FontSize
		sameLine := math.Abs(t.Y-cur.Y) <= yTolerance(cur.FontSize)
		gap := t.X - curRight

		if sameSize && sameLine && gap <= xGap(cur.FontSize) {
			cur.Text += t.S
			if t.X > curRight {
				curRight = t.X
			}
			continue
		}

		flush()
		cur = newBlock(t, pageNum)
		curRight = t.X
	}
	flush()

	return blocks
}

func newBlock(t pdf.Text, pageNum int) *Block {
	return &Block{
		Text:     t.S,
		FontSize: t.FontSize,
		X:        t.X,
		Y:        t.Y,
		Page:     pageNum,
	}
}

func yTolerance(fontSize float64) float64 {
	return fontSize * 0.5
}

func xGap(fontSize float64) float64 {
	return fontSize
}

func estimateWidth(fontSize float64, text string) float64 {
	return fontSize * 0.5 * float64(len([]rune(text)))
}
