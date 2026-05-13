package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/pkg/response"
)

// JWTAuth middleware untuk validasi JWT token dan ekstraksi NPM dari claims
type JWTAuth struct {
	secretKey string
	log       *slog.Logger
}

// NewJWTAuth membuat instance JWTAuth baru
func NewJWTAuth(secretKey string, log *slog.Logger) *JWTAuth {
	return &JWTAuth{
		secretKey: secretKey,
		log:       log,
	}
}

// Handle melakukan validasi JWT token dan mengekstrak NPM dari claims
func (j *JWTAuth) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			j.log.WarnContext(ctx, "Missing Authorization header",
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("path", r.URL.Path),
			)
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Authorization header is required")
			return
		}

		// Check token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			j.log.WarnContext(ctx, "Invalid Authorization header format",
				slog.String("remote_addr", r.RemoteAddr),
			)
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Invalid Authorization header format")
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(j.secretKey), nil
		})

		if err != nil {
			j.log.WarnContext(ctx, "Invalid JWT token", slog.String("error", err.Error()))
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Invalid JWT token")
			return
		}

		if !token.Valid {
			j.log.WarnContext(ctx, "JWT token marked invalid")
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Invalid JWT token")
			return
		}

		// Validate expiration (double-check; jwt.Parse sudah memvalidasi ini,
		// tapi kita tambahkan manual check untuk kejelasan)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			j.log.WarnContext(ctx, "Invalid JWT claims structure")
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Invalid JWT claims")
			return
		}

		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				j.log.WarnContext(ctx, "JWT token expired")
				response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "Token expired")
				return
			}
		}

		// Extract NPM from claims
		npm, ok := claims["npm"].(string)
		if !ok || npm == "" {
			j.log.WarnContext(ctx, "NPM not found in JWT claims")
			response.Error(w, http.StatusUnauthorized, model.ErrCodeUnauthorized, "NPM not found in token claims")
			return
		}

		// Add NPM to context
		ctx = context.WithValue(ctx, userNPMKey, npm)
		j.log.DebugContext(ctx, "JWT authentication successful", slog.String("npm", npm))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
