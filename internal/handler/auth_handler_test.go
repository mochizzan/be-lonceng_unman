package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/services"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupAuthHandler() *AuthHandler {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return NewAuthHandler(log)
}

func TestGenerateToken_Success(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate?npm=2211700006", nil)
	req.Header.Set("X-API-KEY", "test-api-key")
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGenerateToken_MissingAPIKey(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate?npm=2211700006", nil)
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGenerateToken_InvalidAPIKey(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate?npm=2211700006", nil)
	req.Header.Set("X-API-KEY", "wrong-key")
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGenerateToken_MissingNPM(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate", nil)
	req.Header.Set("X-API-KEY", "test-api-key")
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerateToken_InvalidNPM(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate?npm=123", nil)
	req.Header.Set("X-API-KEY", "test-api-key")
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerateToken_TokenValid(t *testing.T) {
	_ = os.Setenv("API_KEY", "test-api-key")
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	_ = config.LoadConfig()

	h := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/generate?npm=2211700006", nil)
	req.Header.Set("X-API-KEY", "test-api-key")
	w := httptest.NewRecorder()

	h.GenerateToken(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the response body can be parsed
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "data field should be a map")

	tokenString, ok := data["token"].(string)
	assert.True(t, ok, "token field should be a string")

	// Verify the JWT token can be parsed and is valid
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key-for-testing"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "2211700006", claims["npm"])
}

// Suppress unused import warning
var _ = services.ValidateNIS
