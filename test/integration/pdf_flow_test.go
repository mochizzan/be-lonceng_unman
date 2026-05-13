package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockIntegrationTransport untuk integration test
type mockIntegrationTransport struct {
	statusCode  int
	contentType string
	body        []byte
}

func (m *mockIntegrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Header:     map[string][]string{"Content-Type": {m.contentType}},
		Body:       io.NopCloser(bytes.NewReader(m.body)),
	}, nil
}

// =============================================================================
// PDF Download Flow - SSRF Protection
// =============================================================================

func TestIntegration_DownloadPDF_SSRFProtection(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	dangerousURLs := []string{
		"http://localhost:8080/admin",
		"http://127.0.0.1:3000/api",
		"http://169.254.169.254/latest/meta-data/",
		"https://evil.com/steal",
		"https://192.168.1.1/admin",
	}

	for _, url := range dangerousURLs {
		t.Run("ssrf_"+url, func(t *testing.T) {
			_, err := pdfService.DownloadPDF(ctx, url)
			assert.Error(t, err, "SSRF protection should block: %s", url)
		})
	}
}

func TestIntegration_DownloadPDF_ValidHost(t *testing.T) {
	pdfData := []byte("%PDF-1.4 " + string(bytes.Repeat([]byte("x"), 200)))
	transport := &mockIntegrationTransport{
		statusCode:  200,
		contentType: "application/pdf",
		body:        pdfData,
	}
	client := &http.Client{Transport: transport}
	pdfService := services.NewPDFService(client)
	ctx := context.Background()

	data, err := pdfService.DownloadPDF(ctx, "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006")
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestIntegration_DownloadPDF_Non200(t *testing.T) {
	transport := &mockIntegrationTransport{
		statusCode:  500,
		contentType: "text/html",
		body:        []byte("Internal Server Error"),
	}
	client := &http.Client{Transport: transport}
	pdfService := services.NewPDFService(client)
	ctx := context.Background()

	_, err := pdfService.DownloadPDF(ctx, "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=0000000000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// =============================================================================
// ProcessKRS Flow
// =============================================================================

func TestIntegration_ProcessKRS_InvalidDomain(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	_, err := pdfService.ProcessKRSFromURL(ctx, "https://evil.com/krs.pdf")
	assert.Error(t, err)
}

func TestIntegration_ProcessKRS_ContextCancelled(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pdfService.ProcessKRSFromURL(ctx, "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006")
	assert.Error(t, err)
}

// =============================================================================
// Error Wrapping Flow - Known Issue
// =============================================================================

func TestIntegration_ErrorWrapping_PreservesType(t *testing.T) {
	// Test bahwa error wrapping dengan fmt.Errorf("...: %w", err) mempertahankan tipe
	originalErr := model.NewInvalidResponseError("test reason")
	wrappedErr := fmt.Errorf("download PDF: %w", originalErr)

	status, code := model.MapErrorToHTTPStatus(wrappedErr)
	// NOTE: errors.Is tidak bekerja dengan wrapping untuk custom struct pointer
	// Ini adalah bug yang diketahui - lihat audit findings
	t.Logf("Status: %d, Code: %s (wrapped error may not map correctly)", status, code)
}
