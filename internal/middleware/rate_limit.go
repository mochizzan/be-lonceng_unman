package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/pkg/response"
)

// RateLimit struct untuk rate limiting dengan konfigurasi dinamis
type RateLimit struct {
	visitors   map[string]*visitor
	mu         sync.RWMutex
	rate       rate.Limit
	burst      int
	log        *slog.Logger
	trustProxy bool
	ctx        context.Context
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimit membuat instance RateLimit baru
func NewRateLimit(r rate.Limit, burst int, log *slog.Logger, trustProxy bool, ctx context.Context) *RateLimit {
	rl := &RateLimit{
		visitors:   make(map[string]*visitor),
		rate:       r,
		burst:      burst,
		log:        log,
		trustProxy: trustProxy,
		ctx:        ctx,
	}

	// Jalankan cleanup goroutine
	go rl.cleanup()

	return rl
}

// getClientIP extracts the client IP address considering trustProxy setting.
// When trustProxy is true, it checks X-Forwarded-For and X-Real-IP headers.
// When trustProxy is false, only r.RemoteAddr is used.
func (rl *RateLimit) getClientIP(r *http.Request) string {
	if rl.trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ips := strings.Split(forwarded, ",")
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			if net.ParseIP(realIP) != nil {
				return realIP
			}
		}
	}
	// Fallback to RemoteAddr (strip port if present)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getLimiter mendapatkan rate limiter untuk IP tertentu
func (rl *RateLimit) getLimiter(r *http.Request) *rate.Limiter {
	ip := rl.getClientIP(r)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// Handle menerapkan rate limiting pada handler
func (rl *RateLimit) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.getLimiter(r).Allow() {
			ip := rl.getClientIP(r)
			rl.log.WarnContext(r.Context(), "Rate limit exceeded", slog.String("ip", ip))
			response.Error(w, http.StatusTooManyRequests, model.ErrCodeRateLimit, "Rate limit exceeded, coba lagi nanti")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cleanup menghapus visitor yang sudah tidak aktif
func (rl *RateLimit) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.ctx.Done():
			// Context cancelled, exit cleanup goroutine
			rl.log.InfoContext(rl.ctx, "Rate limiter cleanup stopped")
			return
		case <-ticker.C:
			now := time.Now()
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}
