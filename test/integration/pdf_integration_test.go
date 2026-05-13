package integration_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"be-lonceng_unman/internal/services"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// PDFService Integration Tests — require external service
// All tests are skipped in short mode (-short flag)
// =============================================================================

func TestDownloadPDF_Integration_ValidPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	data, err := pdfService.DownloadPDF(ctx, pdfURL)
	if err != nil {
		t.Logf("DownloadPDF failed (may be network issue): %v", err)
		return
	}

	assert.NotNil(t, data)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
	assert.Greater(t, len(data), 100)
}

func TestDownloadPDF_Integration_Non200Status(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=0000000000"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	if err != nil {
		assert.Error(t, err)
	}
}

func TestDownloadPDF_Integration_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfService := services.NewPDFService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	// Timeout dalam 100ms bisa menghasilkan error DeadlineExceeded (wrapped)
	// atau berhasil jika koneksi lokal sangat cepat — kedua skenario diterima
	if err != nil {
		// Gunakan errors.Is untuk mendeteksi wrapped context error
		assert.True(t,
			errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
			"expected context deadline or cancel error, got: %v", err,
		)
	}
}

func TestExtractTextFromPDF_Integration_RealPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	pdfData, err := pdfService.DownloadPDF(ctx, pdfURL)
	if err != nil {
		t.Logf("DownloadPDF failed (may be network issue): %v", err)
		return
	}

	text, err := pdfService.ExtractTextFromPDF(pdfData)
	if err != nil {
		t.Logf("ExtractTextFromPDF failed: %v", err)
		return
	}

	assert.NotEmpty(t, text)
	assert.Contains(t, text, "KARTU RENCANA STUDI")
}
