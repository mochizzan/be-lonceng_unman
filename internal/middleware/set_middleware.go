package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

func SetMiddleware(f http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		f = m[i](f)
	}
	return f
}