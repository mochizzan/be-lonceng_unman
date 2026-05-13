package middleware

import "context"

type contextKey struct{}

var userNPMKey contextKey

// GetUserNPM mengambil NPM dari context
func GetUserNPM(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userNPMKey).(string)
	return val, ok
}

// SetUserNPM menambahkan NPM ke context
func SetUserNPM(ctx context.Context, npm string) context.Context {
	return context.WithValue(ctx, userNPMKey, npm)
}
