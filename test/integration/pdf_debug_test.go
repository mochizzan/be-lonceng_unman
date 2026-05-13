package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/services"
)

func TestPDF_DebugExtractText(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pdfService := services.NewPDFService(nil)

	nis := "1234567890"
	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=" + nis

	if err := config.EnsureOutputDir(); err != nil {
		t.Fatalf("Failed to ensure output directory: %v", err)
	}
	outputDir := config.GetOutputDir()

	pdfData, err := pdfService.DownloadPDF(ctx, pdfURL)
	if err != nil {
		t.Fatalf("Failed to download PDF: %v", err)
	}

	t.Logf("PDF downloaded successfully, size: %d bytes", len(pdfData))

	pdfPath := filepath.Join(outputDir, "krs_"+nis+"_raw.pdf")
	os.WriteFile(pdfPath, pdfData, 0644)
	t.Logf("Raw PDF saved to %s", pdfPath)

	textContent, err := pdfService.ExtractTextFromPDF(pdfData)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	t.Logf("Extracted text length: %d characters", len(textContent))

	txtPath := filepath.Join(outputDir, "krs_"+nis+"_extracted_text.txt")
	os.WriteFile(txtPath, []byte(textContent), 0644)
	t.Logf("Extracted text saved to %s", txtPath)

	if len(textContent) > 2000 {
		t.Logf("Extracted text (first 2000 chars):\n%s", textContent[:2000])
	} else {
		t.Logf("Extracted text:\n%s", textContent)
	}
}
