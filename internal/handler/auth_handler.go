package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"be-lonceng_unman/internal/config"
	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/pkg/response"
	"be-lonceng_unman/internal/services"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler menangani autentikasi dan token generation
type AuthHandler struct {
	log *slog.Logger
}

// NewAuthHandler membuat instance AuthHandler baru
func NewAuthHandler(log *slog.Logger) *AuthHandler {
	return &AuthHandler{log: log}
}

// GenerateToken menangani request untuk generate JWT token.
// Endpoint ini membutuhkan X-API-KEY header yang valid.
func (h *AuthHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed, gunakan POST")
		return
	}

	// Validasi API Key dengan constant time compare untuk mencegah timing attack
	apiKey := r.Header.Get("X-API-KEY")
	if apiKey == "" {
		h.log.WarnContext(ctx, "GenerateToken: missing API key",
			slog.String("remote_addr", r.RemoteAddr),
		)
		response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "X-API-KEY header is required")
		return
	}

	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(config.GetAPIKey())) != 1 {
		h.log.WarnContext(ctx, "GenerateToken: invalid API key attempt",
			slog.String("remote_addr", r.RemoteAddr),
		)
		response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Invalid API key")
		return
	}

	// Ambil NPM dari query parameter
	npm := r.URL.Query().Get("npm")
	if npm == "" {
		response.Error(w, http.StatusBadRequest, model.ErrCodeNISInvalid, "NPM parameter is required")
		return
	}

	// Validasi format NPM (10 digit angka)
	if err := services.ValidateNIS(npm); err != nil {
		h.log.WarnContext(ctx, "GenerateToken: invalid NPM format", slog.String("npm", npm))
		response.Error(w, http.StatusBadRequest, model.ErrCodeNISInvalid, "Invalid NPM format: must be 10 digits")
		return
	}

	// Generate JWT token
	expiry := time.Now().Add(1 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"npm": npm,
		"exp": expiry.Unix(),
	})

	tokenString, err := token.SignedString([]byte(config.GetJWTSecret()))
	if err != nil {
		h.log.ErrorContext(ctx, "GenerateToken: failed to sign JWT", slog.String("error", err.Error()))
		response.Error(w, http.StatusInternalServerError, model.ErrCodeInternalServer, "Failed to generate token")
		return
	}

	h.log.InfoContext(ctx, "Token generated successfully", slog.String("npm", npm))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"token":   tokenString,
			"npm":     npm,
			"expires": expiry.UTC().Format(time.RFC3339),
		},
	}); err != nil {
		h.log.ErrorContext(ctx, "GenerateToken: failed to encode response", slog.String("error", err.Error()))
	}
}
