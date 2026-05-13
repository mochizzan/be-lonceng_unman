package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"be-lonceng_unman/internal/handler"
	"be-lonceng_unman/internal/middleware"
	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/services"
	"be-lonceng_unman/internal/storage"
	"be-lonceng_unman/test/mocks"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Helper functions
// =============================================================================

func createTestLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func createValidJWT(npm string, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"npm": npm,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func createExpiredJWT(npm string, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"npm": npm,
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// mockKRSResponsePDFTransport returns a minimal valid PDF for handler flow tests
type mockKRSResponsePDFTransport struct{}

func (m *mockKRSResponsePDFTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	pdfContent := []byte("%PDF-1.4 minimal pdf for handler test")
	return &http.Response{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"application/pdf"}},
		Body:       io.NopCloser(bytes.NewReader(pdfContent)),
	}, nil
}

func setupKRSHandler() *handler.KRSHandler {
	mockClient := &http.Client{Transport: &mockKRSResponsePDFTransport{}}
	pdfService := services.NewPDFService(mockClient)
	cacheService := services.NewCacheService(context.Background())
	fileStorage := storage.NewFileStorage(".")
	log := createTestLogger()
	return handler.NewKRSHandler(pdfService, cacheService, fileStorage, log)
}

// =============================================================================
// Test JWTAuth Middleware
// =============================================================================

func TestJWTAuth_MissingToken(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	req := httptest.NewRequest("GET", "/api/krs", nil)
	w := httptest.NewRecorder()

	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, model.ErrCodeUnauthorized, resp.Data.Code)
}

func TestJWTAuth_InvalidTokenFormat(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()

	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	token := createExpiredJWT("2211700006", secret)

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_MissingNPMClaim(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-secret"
	jwtAuth := middleware.NewJWTAuth(secret, createTestLogger())

	token := createValidJWT("2211700006", secret)

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	var extractedNPM string
	handler := jwtAuth.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedNPM, _ = middleware.GetUserNPM(r.Context())
	}))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2211700006", extractedNPM)
}

// =============================================================================
// Test KRSHandler.GetKRSByNIS - Error Cases
// =============================================================================

func TestGetKRSByNIS_Unauthorized(t *testing.T) {
	h := setupKRSHandler()

	req := httptest.NewRequest("GET", "/api/krs", nil)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, model.ErrCodeUnauthorized, resp.Data.Code)
}

func TestGetKRSByNIS_InvalidNIS(t *testing.T) {
	h := setupKRSHandler()

	ctx := middleware.SetUserNPM(context.Background(), "123")

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, model.ErrCodeNISInvalid, resp.Data.Code)
}

func TestGetKRSByNIS_EmptyNPM(t *testing.T) {
	h := setupKRSHandler()

	ctx := middleware.SetUserNPM(context.Background(), "")

	req := httptest.NewRequest("GET", "/api/krs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =============================================================================
// Test KRSHandler.GetKRSByNIS - Success Cases (with mocked PDF service)
// =============================================================================

func TestGetKRSByNIS_MockPDFService_Success(t *testing.T) {
	mockPDF := mocks.NewMockPDFServiceSuccess("2211700006", "Test User", "TI")
	cacheService := services.NewCacheService(context.Background())
	fileStorage := storage.NewFileStorage(".")
	log := createTestLogger()

	h := handler.NewKRSHandler(mockPDF, cacheService, fileStorage, log)

	ctx := middleware.SetUserNPM(context.Background(), "2211700006")
	req := httptest.NewRequest("GET", "/api/krs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response.Status)
}

func TestGetKRSByNIS_MockPDFService_DownloadError(t *testing.T) {
	mockPDF := mocks.NewMockPDFServiceError(model.NewPDFDownloadFailedError("test-url", nil, 500))
	cacheService := services.NewCacheService(context.Background())
	fileStorage := storage.NewFileStorage(".")
	log := createTestLogger()

	h := handler.NewKRSHandler(mockPDF, cacheService, fileStorage, log)

	ctx := middleware.SetUserNPM(context.Background(), "2211700006")
	req := httptest.NewRequest("GET", "/api/krs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.True(t, w.Code >= 400, "Expected error status code, got %d", w.Code)
}

func TestGetKRSByNIS_MockPDFService_EmptyData(t *testing.T) {
	mockPDF := mocks.NewMockPDFServiceEmptyData()
	cacheService := services.NewCacheService(context.Background())
	fileStorage := storage.NewFileStorage(".")
	log := createTestLogger()

	h := handler.NewKRSHandler(mockPDF, cacheService, fileStorage, log)

	ctx := middleware.SetUserNPM(context.Background(), "2211700006")
	req := httptest.NewRequest("GET", "/api/krs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKRSByNIS(w, req)

	assert.True(t, w.Code >= 400, "Expected error status code for empty data, got %d", w.Code)
}

// =============================================================================
// Test Response Format
// =============================================================================

func TestResponseFormat_Success(t *testing.T) {
	response := model.SuccessResponse{
		Status: "success",
		Data: model.KRSResponse{
			Status:      "success",
			Mahasiswa:   model.Mahasiswa{NPM: "2211700006", Nama: "Test"},
			TahunAjaran: "2025/2026",
			Semester:    "GENAP",
			MataKuliah:  []model.MataKuliah{},
			TotalSKS:    0,
		},
		Meta: model.ResponseMeta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "1.0.0",
			Cached:    false,
		},
	}

	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "success", parsed["status"])
	assert.NotNil(t, parsed["data"])
	assert.NotNil(t, parsed["meta"])

	meta := parsed["meta"].(map[string]interface{})
	assert.NotEmpty(t, meta["timestamp"])
	assert.Equal(t, "1.0.0", meta["version"])
	// cached selalu ada di response (bukan omitempty)
	cachedVal, hasCached := meta["cached"]
	assert.True(t, hasCached, "meta.cached should always be present")
	assert.Equal(t, false, cachedVal)
}

func TestResponseFormat_Error(t *testing.T) {
	response := model.ErrorResponse{
		Status:    "error",
		Code:      "ERR_NIS_INVALID",
		Message:   "Invalid NIS format",
		Details:   "NIS must be 10 digits",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "error", parsed["status"])
	assert.Equal(t, "ERR_NIS_INVALID", parsed["code"])
	assert.Equal(t, "Invalid NIS format", parsed["message"])
	assert.Equal(t, "NIS must be 10 digits", parsed["details"])
	assert.NotEmpty(t, parsed["timestamp"])
}
