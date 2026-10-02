package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-pdf/fpdf"
)

func generateFixtures(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := genSimple(filepath.Join(dir, "simple.pdf")); err != nil {
		return err
	}
	if err := genTable(filepath.Join(dir, "table.pdf")); err != nil {
		return err
	}
	if err := genScanned(filepath.Join(dir, "scanned.pdf")); err != nil {
		return err
	}
	if err := genCorrupt(filepath.Join(dir, "corrupt.pdf")); err != nil {
		return err
	}

	return nil
}

func genSimple(path string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Helvetica", "", 24)
	pdf.SetXY(20, 20)
	pdf.CellFormat(0, 10, "Titulo principal", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 16)
	pdf.SetXY(20, 34)
	pdf.CellFormat(0, 8, "Subtitulo de la seccion", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 12)
	pdf.SetXY(20, 46)
	pdf.CellFormat(0, 6, "Parrafo de texto normal con contenido.", "", 1, "L", false, 0, "")

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("simple.pdf: %w", err)
	}
	return nil
}

func genTable(path string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)

	headers := []string{"Col A", "Col B", "Col C"}
	cols := []float64{20, 70, 120}
	rows := []float64{30, 42, 54}
	data := [][]string{
		{"a1", "b1", "c1"},
		{"a2", "b2", "c2"},
	}

	for c, h := range headers {
		pdf.SetXY(cols[c], rows[0])
		pdf.CellFormat(40, 8, h, "1", 0, "L", false, 0, "")
	}
	for r := 1; r <= 2; r++ {
		for c := 0; c < 3; c++ {
			pdf.SetXY(cols[c], rows[r])
			pdf.CellFormat(40, 8, data[r-1][c], "1", 0, "L", false, 0, "")
		}
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("table.pdf: %w", err)
	}
	return nil
}

func genScanned(path string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.5)
	pdf.Rect(20, 20, 80, 40, "D")
	pdf.Line(20, 80, 100, 120)

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("scanned.pdf: %w", err)
	}
	return nil
}

func genCorrupt(path string) error {
	data := []byte("%PDF-1.4\n% este archivo esta truncado a proposito\n1 0 obj\n<<>>\nstream\n")

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("corrupt.pdf: %w", err)
	}
	return nil
}
