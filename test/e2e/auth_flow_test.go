package e2e_test

import (
	"context"
	"encoding/json"
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
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func setupE2EHandler() *handler.KRSHandler {
	mockPDF := mocks.NewMockPDFServiceSuccess("2211700006", "TEST USER", "TI")
	cacheService := services.NewCacheService(context.Background())
	fileStorage := storage.NewFileStorage(".")
	return handler.NewKRSHandler(mockPDF, cacheService, fileStorage, testLogger())
}

func createTestJWT(secret, npm string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"npm": npm,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestE2E_AuthFlow_NoJWT(t *testing.T) {
	h := setupE2EHandler()
	jwtAuth := middleware.NewJWTAuth("test-secret", testLogger())

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(h.GetKRSByNIS)))

	req := httptest.NewRequest("GET", "/krs", nil)
	w := httptest.NewRecorder()

	protectedMux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestE2E_AuthFlow_InvalidJWT(t *testing.T) {
	h := setupE2EHandler()
	jwtAuth := middleware.NewJWTAuth("test-secret", testLogger())

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(h.GetKRSByNIS)))

	req := httptest.NewRequest("GET", "/krs", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	protectedMux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestE2E_AuthFlow_ValidJWT_Success(t *testing.T) {
	h := setupE2EHandler()
	jwtAuth := middleware.NewJWTAuth("test-secret", testLogger())

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(h.GetKRSByNIS)))

	tokenString := createTestJWT("test-secret", "2211700006")

	req := httptest.NewRequest("GET", "/krs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	protectedMux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var response model.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response.Status)
}

func TestE2E_AuthFlow_InvalidNIS(t *testing.T) {
	h := setupE2EHandler()
	jwtAuth := middleware.NewJWTAuth("test-secret", testLogger())

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(h.GetKRSByNIS)))

	tokenString := createTestJWT("test-secret", "123")

	req := httptest.NewRequest("GET", "/krs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	protectedMux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestE2E_AuthFlow_MissingNPMClaim(t *testing.T) {
	h := setupE2EHandler()
	jwtAuth := middleware.NewJWTAuth("test-secret", testLogger())

	protectedMux := http.NewServeMux()
	protectedMux.Handle("/krs", jwtAuth.Handle(http.HandlerFunc(h.GetKRSByNIS)))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))

	req := httptest.NewRequest("GET", "/krs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	protectedMux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
