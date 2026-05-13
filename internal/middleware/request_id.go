package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKeyRequestID struct{}

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestIDMiddleware middleware yang menambahkan request ID ke setiap request.
// Jika header X-Request-Id sudah ada, gunakan nilai tersebut.
// Jika tidak, generate request ID baru.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = generateRequestID()
		}
		ctx := context.WithValue(r.Context(), contextKeyRequestID{}, requestID)
		w.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID mengambil request ID dari context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(contextKeyRequestID{}).(string); ok {
		return id
	}
	return ""
}
