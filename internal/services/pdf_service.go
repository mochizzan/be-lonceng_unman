package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/parser"

	"github.com/ledongthuc/pdf"
)

// PDFService menangani pengunduhan dan pemrosesan PDF KRS
type PDFService struct {
	client *http.Client
}

// NewPDFService membuat instance PDFService baru dengan custom HTTP client.
// Jika client nil, akan dibuat default dengan timeout 30 detik.
func NewPDFService(client *http.Client) *PDFService {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &PDFService{
		client: client,
	}
}

// DownloadPDF mengunduh PDF dari URL eksternal
func (s *PDFService) DownloadPDF(ctx context.Context, urlStr string) ([]byte, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "https" {
		return nil, model.NewInvalidResponseError("only HTTPS URLs are allowed")
	}

	if u.Hostname() == "" {
		return nil, model.NewInvalidResponseError("URL must have a hostname")
	}

	allowedHosts := config.GetAllowedPDFHosts()
	allowed := false
	for _, host := range allowedHosts {
		if u.Hostname() == host {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, model.NewInvalidResponseError(fmt.Sprintf("URL host not allowed: %s", u.Hostname()))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, model.NewPDFDownloadFailedError(urlStr, err, 0)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, model.NewPDFDownloadFailedError(urlStr, nil, resp.StatusCode)
	}

	// Validasi content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		return nil, model.NewInvalidResponseError("invalid content type: not a PDF")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Validate PDF content
	if len(data) < 100 {
		return nil, model.NewInvalidResponseError("PDF file too small, likely an error page")
	}

	// Check PDF magic bytes
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, model.NewInvalidResponseError("invalid PDF file: missing PDF header")
	}

	return data, nil
}

// ExtractTextFromPDF mengekstrak teks dari PDF menggunakan ledongthuc/pdf
func (s *PDFService) ExtractTextFromPDF(pdfData []byte) (string, error) {
	if len(pdfData) == 0 {
		return "", errors.New("PDF data is empty")
	}

	// Baca PDF dari bytes
	reader, err := pdf.NewReader(bytes.NewReader(pdfData), int64(len(pdfData)))
	if err != nil {
		return "", fmt.Errorf("create PDF reader: %w", err)
	}

	var textBuilder strings.Builder
	totalPages := reader.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := reader.Page(pageNum)
		// Skip invalid pages - ledongthuc/pdf doesn't provide direct page validation
		// We'll proceed and let the GetPlainText method handle errors

		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("extract text from page %d: %w", pageNum, err)
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	if textBuilder.Len() == 0 {
		return "", model.NewInvalidResponseError("no text extracted from PDF")
	}

	return textBuilder.String(), nil
}

// ProcessKRSFromURL mengunduh PDF dari URL, mengekstrak teks, dan memparsing menjadi struktur KRS
func (s *PDFService) ProcessKRSFromURL(ctx context.Context, pdfURL string) (*model.KRSResponse, error) {
	// 1. Download PDF
	pdfData, err := s.DownloadPDF(ctx, pdfURL)
	if err != nil {
		return nil, fmt.Errorf("download PDF: %w", err)
	}

	// 2. Extract text from PDF
	textContent, err := s.ExtractTextFromPDF(pdfData)
	if err != nil {
		return nil, fmt.Errorf("extract text from PDF: %w", err)
	}

	// 3. Parse text to KRS structure
	parser := parser.NewKRSParser(textContent)
	krsResponse, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse KRS text: %w", err)
	}

	return krsResponse, nil
}

// ValidateNIS memvalidasi format NIS (Nomor Induk Siswa)
func ValidateNIS(nis string) error {
	if nis == "" {
		return model.NewNISEmptyError()
	}

	// Format NIS: 10 digit angka
	matched, err := regexp.MatchString(`^\d{10}$`, nis)
	if err != nil || !matched {
		return model.NewNISInvalidError(nis)
	}

	return nil
}
