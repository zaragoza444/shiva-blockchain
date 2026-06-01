package api

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type middleware struct {
	corsOrigins []string
	apiKey      string
	limiter     *rateLimiter
}

func newMiddleware(corsOrigins []string, apiKey string) *middleware {
	return &middleware{
		corsOrigins: corsOrigins,
		apiKey:      apiKey,
		limiter:     newRateLimiter(120, time.Minute),
	}
}

func (m *middleware) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.setSecurityHeaders(w)
		if m.handleCORS(w, r) {
			return
		}
		if !m.limiter.allow(remoteIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if m.apiKey != "" && needsAPIKey(r) {
			if r.Header.Get("X-OneX-Api-Key") != m.apiKey && r.Header.Get("Authorization") != "Bearer "+m.apiKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func (m *middleware) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	if len(m.corsOrigins) == 0 || originAllowed(origin, m.corsOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-OneX-Api-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func originAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == "*" || strings.EqualFold(strings.TrimSpace(o), origin) {
			return true
		}
	}
	return false
}

func needsAPIKey(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case "/api/v1/tx", "/api/v1/faucet", "/rpc":
		return true
	default:
		return false
	}
}

func remoteIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}
	if x := r.Header.Get("X-Real-IP"); x != "" {
		return strings.TrimSpace(x)
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counts   map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, counts: make(map[string][]time.Time)}
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	var kept []time.Time
	for _, t := range rl.counts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.limit {
		rl.counts[key] = kept
		return false
	}
	kept = append(kept, now)
	rl.counts[key] = kept
	return true
}
