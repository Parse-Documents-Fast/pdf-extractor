package markdown

import (
	"math"
	"sort"
	"strings"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/extractor"
)

type Options struct {
	HeadingRatioH1 float64
	HeadingRatioH2 float64
	HeadingRatioH3 float64
}

func DefaultOptions() Options {
	return Options{
		HeadingRatioH1: 1.8,
		HeadingRatioH2: 1.4,
		HeadingRatioH3: 1.15,
	}
}

func MapStructureToMarkdown(blocks []extractor.Block, opts Options) string {
	opts = normalizeOptions(opts)
	if len(blocks) == 0 {
		return ""
	}

	pred := predominantSize(blocks)
	rows := groupRows(blocks)

	var out []string
	i := 0
	for i < len(rows) {
		if end, ok := tableRun(rows, i); ok {
			out = append(out, renderTable(rows[i:end]))
			i = end
			continue
		}

		for _, b := range rows[i].blocks {
			if s := renderBlock(b, pred, opts); s != "" {
				out = append(out, s)
			}
		}
		i++
	}

	return strings.Join(out, "\n\n") + "\n"
}

func normalizeOptions(opts Options) Options {
	if opts.HeadingRatioH1 <= 0 {
		opts.HeadingRatioH1 = 1.8
	}
	if opts.HeadingRatioH2 <= 0 {
		opts.HeadingRatioH2 = 1.4
	}
	if opts.HeadingRatioH3 <= 0 {
		opts.HeadingRatioH3 = 1.15
	}
	return opts
}

func predominantSize(blocks []extractor.Block) float64 {
	counts := map[float64]int{}
	for _, b := range blocks {
		counts[b.FontSize]++
	}

	var best float64
	bestCount := -1
	for size, c := range counts {
		if c > bestCount || (c == bestCount && size < best) {
			best = size
			bestCount = c
		}
	}
	return best
}

func headingLevel(size, pred float64, opts Options) int {
	if pred <= 0 {
		return 0
	}
	r := size / pred
	switch {
	case r >= opts.HeadingRatioH1:
		return 1
	case r >= opts.HeadingRatioH2:
		return 2
	case r >= opts.HeadingRatioH3:
		return 3
	default:
		return 0
	}
}

func renderBlock(b extractor.Block, pred float64, opts Options) string {
	text := strings.TrimSpace(b.Text)
	if text == "" {
		return ""
	}
	escaped := escapeMarkdown(text)
	switch headingLevel(b.FontSize, pred, opts) {
	case 1:
		return "# " + escaped
	case 2:
		return "## " + escaped
	case 3:
		return "### " + escaped
	default:
		return escaped
	}
}

type row struct {
	page   int
	y      float64
	blocks []extractor.Block
}

func groupRows(blocks []extractor.Block) []row {
	sorted := append([]extractor.Block(nil), blocks...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Page != sorted[j].Page {
			return sorted[i].Page < sorted[j].Page
		}
		if sorted[i].Y != sorted[j].Y {
			return sorted[i].Y > sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})

	var rows []row
	for _, b := range sorted {
		if len(rows) == 0 ||
			rows[len(rows)-1].page != b.Page ||
			math.Abs(b.Y-rows[len(rows)-1].y) > 1.0 {
			rows = append(rows, row{page: b.Page, y: b.Y, blocks: []extractor.Block{b}})
			continue
		}
		rows[len(rows)-1].blocks = append(rows[len(rows)-1].blocks, b)
	}
	return rows
}

func tableRun(rows []row, start int) (int, bool) {
	if start >= len(rows) || len(rows[start].blocks) < 2 {
		return start, false
	}

	ncols := len(rows[start].blocks)
	xs := columnXs(rows[start])

	end := start
	for end < len(rows) && len(rows[end].blocks) >= 2 {
		if rows[end].page != rows[start].page {
			break
		}
		if len(rows[end].blocks) != ncols {
			break
		}
		if !aligned(columnXs(rows[end]), xs) {
			break
		}
		end++
	}

	if end-start < 2 {
		return start, false
	}
	return end, true
}

func columnXs(r row) []float64 {
	xs := make([]float64, len(r.blocks))
	for i, b := range r.blocks {
		xs[i] = b.X
	}
	return xs
}

func aligned(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1.0 {
			return false
		}
	}
	return true
}

func renderTable(rows []row) string {
	ncols := len(rows[0].blocks)
	lines := make([]string, 0, len(rows)+1)

	lines = append(lines, tableRow(rows[0].blocks))

	seps := make([]string, ncols)
	for i := range seps {
		seps[i] = "---"
	}
	lines = append(lines, "| "+strings.Join(seps, " | ")+" |")

	for _, r := range rows[1:] {
		lines = append(lines, tableRow(r.blocks))
	}

	return strings.Join(lines, "\n")
}

func tableRow(blocks []extractor.Block) string {
	cells := make([]string, len(blocks))
	for i, b := range blocks {
		cells[i] = escapeCell(b.Text)
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func escapeCell(s string) string {
	return escapeMarkdown(strings.Join(strings.Fields(s), " "))
}

func escapeMarkdown(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '!', '|':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
