package extractor_test

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestExtractSimple(t *testing.T) {
	blocks, err := extractor.ExtractStructure(context.Background(), readFixture(t, "simple.pdf"))
	if err != nil {
		t.Fatalf("ExtractStructure() error = %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("len(blocks) = %d, want 3", len(blocks))
	}

	want := []struct {
		size float64
		text string
	}{
		{24, "Titulo principal"},
		{16, "Subtitulo de la seccion"},
		{12, "Parrafo de texto normal con contenido."},
	}
	for i, w := range want {
		if !closeEnough(blocks[i].FontSize, w.size, 0.1) {
			t.Errorf("blocks[%d].FontSize = %v, want %v", i, blocks[i].FontSize, w.size)
		}
		if blocks[i].Text != w.text {
			t.Errorf("blocks[%d].Text = %q, want %q", i, blocks[i].Text, w.text)
		}
		if blocks[i].Page != 1 {
			t.Errorf("blocks[%d].Page = %d, want 1", i, blocks[i].Page)
		}
	}
}

func TestExtractTable(t *testing.T) {
	blocks, err := extractor.ExtractStructure(context.Background(), readFixture(t, "table.pdf"))
	if err != nil {
		t.Fatalf("ExtractStructure() error = %v", err)
	}
	if len(blocks) != 9 {
		t.Fatalf("len(blocks) = %d, want 9", len(blocks))
	}

	var xs, ys []float64
	for _, b := range blocks {
		if !closeEnough(b.FontSize, 10, 0.1) {
			t.Errorf("FontSize = %v, want 10", b.FontSize)
		}
		xs = append(xs, b.X)
		ys = append(ys, b.Y)
	}

	if got := countDistinct(xs, 0.5); got != 3 {
		t.Errorf("distinct X = %d, want 3", got)
	}
	if got := countDistinct(ys, 0.5); got != 3 {
		t.Errorf("distinct Y = %d, want 3", got)
	}
}

func TestExtractScannedNoText(t *testing.T) {
	_, err := extractor.ExtractStructure(context.Background(), readFixture(t, "scanned.pdf"))
	if !errors.Is(err, extractor.ErrNoExtractableText) {
		t.Fatalf("error = %v, want ErrNoExtractableText", err)
	}
}

func TestExtractCorrupt(t *testing.T) {
	_, err := extractor.ExtractStructure(context.Background(), readFixture(t, "corrupt.pdf"))
	if !errors.Is(err, extractor.ErrCorruptPDF) {
		t.Fatalf("error = %v, want ErrCorruptPDF", err)
	}

	var pe *extractor.PdfExtractionError
	if !errors.As(err, &pe) {
		t.Fatalf("error %v should be *PdfExtractionError", err)
	}
}

func TestExtractEmptyInput(t *testing.T) {
	_, err := extractor.ExtractStructure(context.Background(), nil)
	if !errors.Is(err, extractor.ErrCorruptPDF) {
		t.Fatalf("error = %v, want ErrCorruptPDF", err)
	}
}

func TestExtractCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := extractor.ExtractStructure(ctx, readFixture(t, "simple.pdf"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestPdfExtractionErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	pe := &extractor.PdfExtractionError{Op: "op", Err: inner}
	if !errors.Is(pe, inner) {
		t.Errorf("errors.Is(pe, inner) = false, want true")
	}
}

func closeEnough(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func countDistinct(vals []float64, tol float64) int {
	var clusters []float64
	for _, v := range vals {
		found := false
		for _, c := range clusters {
			if closeEnough(v, c, tol) {
				found = true
				break
			}
		}
		if !found {
			clusters = append(clusters, v)
		}
	}
	return len(clusters)
}
