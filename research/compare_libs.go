package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ledongthuc/pdf"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "gen" {
		if err := generateFixtures("../testdata"); err != nil {
			fmt.Fprintln(os.Stderr, "gen fixtures:", err)
			os.Exit(1)
		}
		fmt.Println("fixtures generados en ../testdata")
		return
	}

	dir := "../testdata"
	for _, name := range []string{"simple.pdf", "table.pdf", "scanned.pdf", "corrupt.pdf"} {
		fmt.Printf("\n===== %s =====\n", name)
		compare(dir, name)
	}
}

func compare(dir, name string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  PANIC (requiere recover() en produccion): %v\n", r)
		}
	}()

	path := filepath.Join(dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  read error: %v\n", err)
		return
	}

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		fmt.Printf("  NewReader error: %v\n", err)
		return
	}

	fmt.Printf("  paginas: %d\n", r.NumPage())
	for p := 1; p <= r.NumPage(); p++ {
		content := r.Page(p).Content()
		fmt.Printf("  pagina %d: %d textos, %d rects\n", p, len(content.Text), len(content.Rect))
		for _, t := range content.Text {
			fmt.Printf("    [font=%s size=%.2f x=%.2f y=%.2f w=%.2f] %q\n", t.Font, t.FontSize, t.X, t.Y, t.W, t.S)
		}
	}
}
