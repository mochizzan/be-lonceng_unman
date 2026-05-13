package unit_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// mockPDFTransport — single shared mock transport (F-03 fix: removed duplicate set)
// =============================================================================

type mockPDFTransport struct {
	statusCode  int
	contentType string
	body        []byte
}

func (m *mockPDFTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Header:     map[string][]string{"Content-Type": {m.contentType}},
		Body:       io.NopCloser(bytes.NewReader(m.body)),
	}, nil
}

func newMockPDFService(statusCode int, contentType string, body []byte) *services.PDFService {
	transport := &mockPDFTransport{
		statusCode:  statusCode,
		contentType: contentType,
		body:        body,
	}
	return services.NewPDFService(&http.Client{Transport: transport})
}

// mockSlowTransport simulates a slow/hanging HTTP response, respects context cancellation
type mockSlowTransport struct {
	delay time.Duration
}

func (m *mockSlowTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()

	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-timer.C:
		return &http.Response{
			StatusCode: 200,
			Header:     map[string][]string{"Content-Type": {"application/pdf"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("%PDF-1.4"))),
		}, nil
	}
}

// =============================================================================
// Test PDFService.DownloadPDF - Domain Validation Tests
// =============================================================================

func TestDownloadPDF_InvalidURL_NonHTTPS(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	_, err := pdfService.DownloadPDF(ctx, "http://example.com/test.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTPS")
}

func TestDownloadPDF_InvalidURL_WrongDomain(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	_, err := pdfService.DownloadPDF(ctx, "https://malicious.com/admin/cetak/krs_pdf.php?nis=2211700006")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "allow")
}

func TestDownloadPDF_NetworkError(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	_, err := pdfService.DownloadPDF(ctx, "https://192.0.2.1/test.pdf")
	assert.Error(t, err)
}

// =============================================================================
// Test PDFService.DownloadPDF - Mock Transport Tests
// =============================================================================

func TestDownloadPDF_MockTransport_Success(t *testing.T) {
	pdfData := []byte("%PDF-1.4 %EOF this is a minimal valid PDF content for testing purposes with enough bytes to pass validation")
	pdfService := newMockPDFService(200, "application/pdf", pdfData)
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	data, err := pdfService.DownloadPDF(ctx, pdfURL)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestDownloadPDF_MockTransport_Non200Status(t *testing.T) {
	pdfService := newMockPDFService(404, "text/html", []byte("Not Found"))
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=0000000000"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownloadPDF_MockTransport_InvalidContentType(t *testing.T) {
	pdfService := newMockPDFService(200, "text/html", []byte("<html>Not a PDF</html>"))
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content type")
}

func TestDownloadPDF_MockTransport_TooSmall(t *testing.T) {
	pdfService := newMockPDFService(200, "application/pdf", []byte("tiny"))
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too small")
}

func TestDownloadPDF_MockTransport_InvalidPDFHeader(t *testing.T) {
	pdfService := newMockPDFService(200, "application/pdf", bytes.Repeat([]byte("A"), 150))
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PDF header")
}

func TestDownloadPDF_MockTransport_ServerError(t *testing.T) {
	pdfService := newMockPDFService(500, "text/html", []byte("Internal Server Error"))
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestDownloadPDF_MockTransport_ContextTimeout(t *testing.T) {
	slowTransport := &mockSlowTransport{
		delay: 5 * time.Second,
	}
	customClient := &http.Client{
		Transport: slowTransport,
	}
	pdfService := services.NewPDFService(customClient)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.DownloadPDF(ctx, pdfURL)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context"),
		"expected context error, got: %v", err)
}

// =============================================================================
// Test PDFService.ExtractTextFromPDF
// =============================================================================

func TestExtractTextFromPDF_EmptyData(t *testing.T) {
	pdfService := services.NewPDFService(nil)

	_, err := pdfService.ExtractTextFromPDF([]byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestExtractTextFromPDF_InvalidPDF(t *testing.T) {
	pdfService := services.NewPDFService(nil)

	_, err := pdfService.ExtractTextFromPDF([]byte("not a pdf"))
	assert.Error(t, err)
}

// =============================================================================
// Test PDFService.ProcessKRSFromURL - Mock Transport Tests
// =============================================================================

func TestProcessKRSFromURL_MockTransport_Success(t *testing.T) {
	pdfData := []byte("%PDF-1.4 this is a longer PDF content that has enough bytes to pass the minimum size validation check but does not contain any real KRS data for parsing into mahasiswa information")
	pdfService := newMockPDFService(200, "application/pdf", pdfData)
	ctx := context.Background()

	pdfURL := "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006"

	_, err := pdfService.ProcessKRSFromURL(ctx, pdfURL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extract text")
}

func TestProcessKRSFromURL_InvalidURL(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx := context.Background()

	_, err := pdfService.ProcessKRSFromURL(ctx, "https://malicious.com/test.pdf")
	assert.Error(t, err)
}

func TestProcessKRSFromURL_ContextCanceled(t *testing.T) {
	pdfService := services.NewPDFService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pdfService.ProcessKRSFromURL(ctx, "https://elearning.universitasmandiri.ac.id/admin/cetak/krs_pdf.php?nis=2211700006")
	assert.Error(t, err)
}

// =============================================================================
// Test ValidateNIS
// =============================================================================

func TestValidateNIS_Standalone(t *testing.T) {
	tests := []struct {
		name    string
		nis     string
		wantErr bool
		errMsg  string
	}{
		{name: "valid NIS - 2211700006", nis: "2211700006", wantErr: false},
		{name: "valid NIS - 1234567890", nis: "1234567890", wantErr: false},
		{name: "valid NIS - 0000000000", nis: "0000000000", wantErr: false},
		{name: "invalid NIS - empty", nis: "", wantErr: true, errMsg: "NIS cannot be empty"},
		{name: "invalid NIS - 9 digits", nis: "221170000", wantErr: true, errMsg: "NIS must be 10 digits"},
		{name: "invalid NIS - 11 digits", nis: "22117000061", wantErr: true, errMsg: "NIS must be 10 digits"},
		{name: "invalid NIS - contains letters", nis: "221170000a", wantErr: true, errMsg: "NIS must be 10 digits"},
		{name: "invalid NIS - contains symbols", nis: "221170000!", wantErr: true, errMsg: "NIS must be 10 digits"},
		{name: "invalid NIS - all letters", nis: "abcdefghij", wantErr: true, errMsg: "NIS must be 10 digits"},
		{name: "invalid NIS - mixed", nis: "abc1234567", wantErr: true, errMsg: "NIS must be 10 digits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := services.ValidateNIS(tt.nis)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Test CacheService
// =============================================================================

func TestCacheService_SetInvalidTTL(t *testing.T) {
	cache := services.NewCacheService(context.Background())
	ctx := context.Background()

	krsResponse := &model.KRSResponse{
		Status:      "success",
		Mahasiswa:   model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
		TahunAjaran: "2025/2026",
		Semester:    "GENAP",
		MataKuliah:  []model.MataKuliah{{No: 1, Kode: "SI40306", Nama: "TEST", SKS: 6}},
		TotalSKS:    6,
	}

	err := cache.Set(ctx, "test:ttl:zero", krsResponse, 0)
	assert.Error(t, err)

	err = cache.Set(ctx, "test:ttl:neg", krsResponse, -1*time.Hour)
	assert.Error(t, err)
}

func TestCacheService_Delete(t *testing.T) {
	cache := services.NewCacheService(context.Background())
	ctx := context.Background()

	krsResponse := &model.KRSResponse{
		Status:      "success",
		Mahasiswa:   model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
		TahunAjaran: "2025/2026",
		Semester:    "GENAP",
		MataKuliah:  []model.MataKuliah{{No: 1, Kode: "SI40306", Nama: "TEST", SKS: 6}},
		TotalSKS:    6,
	}

	err := cache.Set(ctx, "test:delete", krsResponse, 1*time.Hour)
	assert.NoError(t, err)

	err = cache.Delete(ctx, "test:delete")
	assert.NoError(t, err)

	_, err = cache.Get(ctx, "test:delete")
	assert.Error(t, err)
}

func TestCacheService_ExpiredData(t *testing.T) {
	cache := services.NewCacheService(context.Background())
	ctx := context.Background()

	krsResponse := &model.KRSResponse{
		Status:      "success",
		Mahasiswa:   model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
		TahunAjaran: "2025/2026",
		Semester:    "GENAP",
		MataKuliah:  []model.MataKuliah{{No: 1, Kode: "SI40306", Nama: "TEST", SKS: 6}},
		TotalSKS:    6,
	}

	err := cache.Set(ctx, "test:expired", krsResponse, 1*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = cache.Get(ctx, "test:expired")
	assert.Error(t, err)
}
