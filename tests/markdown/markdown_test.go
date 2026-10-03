package markdown_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/markdown"
)

func TestMapHeadings(t *testing.T) {
	blocks := []extractor.Block{
		{Text: "Big", FontSize: 20, X: 10, Y: 100, Page: 1},
		{Text: "Mid", FontSize: 15, X: 10, Y: 80, Page: 1},
		{Text: "Small", FontSize: 12, X: 10, Y: 60, Page: 1},
		{Text: "Body", FontSize: 10, X: 10, Y: 40, Page: 1},
	}

	got := markdown.MapStructureToMarkdown(blocks, markdown.DefaultOptions())
	want := "# Big\n\n## Mid\n\n### Small\n\nBody\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestMapEscapesSpecialChars(t *testing.T) {
	blocks := []extractor.Block{
		{Text: "a * b _ c # d | e `f`", FontSize: 10, X: 0, Y: 100, Page: 1},
	}

	got := markdown.MapStructureToMarkdown(blocks, markdown.DefaultOptions())
	want := "a \\* b \\_ c \\# d \\| e \\`f\\`\n"
	if got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestMapTable(t *testing.T) {
	blocks := []extractor.Block{
		{Text: "H1", FontSize: 10, X: 10, Y: 100, Page: 1},
		{Text: "H2", FontSize: 10, X: 60, Y: 100, Page: 1},
		{Text: "a|b", FontSize: 10, X: 10, Y: 80, Page: 1},
		{Text: "c", FontSize: 10, X: 60, Y: 80, Page: 1},
	}

	got := markdown.MapStructureToMarkdown(blocks, markdown.DefaultOptions())
	want := "| H1 | H2 |\n| --- | --- |\n| a\\|b | c |\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestMapEmpty(t *testing.T) {
	got := markdown.MapStructureToMarkdown(nil, markdown.DefaultOptions())
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestMapCustomThresholds(t *testing.T) {
	blocks := []extractor.Block{
		{Text: "Title", FontSize: 13, X: 10, Y: 100, Page: 1},
		{Text: "Body", FontSize: 10, X: 10, Y: 80, Page: 1},
	}

	opts := markdown.Options{HeadingRatioH1: 1.5, HeadingRatioH2: 1.3, HeadingRatioH3: 1.1}
	got := markdown.MapStructureToMarkdown(blocks, opts)
	// 13/10 = 1.3 => H2
	want := "## Title\n\nBody\n"
	if got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestMapSimplePDF(t *testing.T) {
	blocks := extractFixture(t, "simple.pdf")
	got := markdown.MapStructureToMarkdown(blocks, markdown.DefaultOptions())
	want := "# Titulo principal\n\n### Subtitulo de la seccion\n\nParrafo de texto normal con contenido.\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestMapTablePDF(t *testing.T) {
	blocks := extractFixture(t, "table.pdf")
	got := markdown.MapStructureToMarkdown(blocks, markdown.DefaultOptions())
	want := "| Col A | Col B | Col C |\n| --- | --- | --- |\n| a1 | b1 | c1 |\n| a2 | b2 | c2 |\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func extractFixture(t *testing.T, name string) []extractor.Block {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	blocks, err := extractor.ExtractStructure(context.Background(), data)
	if err != nil {
		t.Fatalf("ExtractStructure(%s): %v", name, err)
	}
	return blocks
}
