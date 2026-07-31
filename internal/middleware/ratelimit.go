package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter enforces per-IP token-bucket rate limits with idle eviction.
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor

	globalLimit   rate.Limit
	globalBurst   int
	sweepInterval time.Duration
	idleTTL       time.Duration
	trustedCIDRs  []*net.IPNet
	logger        *slog.Logger
}

// NewRateLimiter creates a RateLimiter from application config. It starts a
// background goroutine that evicts visitors idle longer than 5 minutes.
func NewRateLimiter(globalRPM int, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		visitors:      make(map[string]*visitor),
		globalLimit:   rate.Limit(float64(globalRPM) / 60.0),
		globalBurst:   globalRPM / 6, // burst = 1/6 of RPM, minimum 1
		sweepInterval: 60 * time.Second,
		idleTTL:       5 * time.Minute,
		logger:        logger,
	}
	if rl.globalBurst < 1 {
		rl.globalBurst = 1
	}
	go rl.sweep()
	return rl
}

// SetTrustedCIDRs configures the list of trusted proxy CIDRs for real-IP
// extraction. Each string must be a valid CIDR (e.g. "10.0.0.0/8").
func (rl *RateLimiter) SetTrustedCIDRs(cidrs []string) {
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			rl.trustedCIDRs = append(rl.trustedCIDRs, n)
		}
	}
}

// Middleware returns a handler that enforces per-IP rate limits. It first
// checks the global limit from config, then the route-specific limit from the
// arguments. A request is rejected if either limit is exceeded.
func (rl *RateLimiter) Middleware(routeLimit, routeBurst int) func(http.Handler) http.Handler {
	routeRate := rate.Limit(float64(routeLimit) / 60.0)
	if routeBurst < 1 {
		routeBurst = 1
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r, rl.trustedCIDRs)

			// Global limit (scope prefix avoids key collisions with route limiters).
			gKey := "global:" + ip
			gl := rl.getLimiter(gKey, rl.globalLimit, rl.globalBurst)
			if !gl.Allow() {
				rl.logger.Warn("rate limit exceeded (global)",
					"ip", ip,
					"path", r.URL.Path,
					"request_id", GetRequestID(r.Context()),
				)
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}

			// Route-specific limit.
			rKey := "route:" + ip
			routeLim := rl.getLimiter(rKey, routeRate, routeBurst)
			if !routeLim.Allow() {
				rl.logger.Warn("rate limit exceeded (route)",
					"ip", ip,
					"path", r.URL.Path,
					"request_id", GetRequestID(r.Context()),
				)
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getLimiter(ip string, limit rate.Limit, burst int) *rate.Limiter {
	rl.mu.RLock()
	v, ok := rl.visitors[ip]
	rl.mu.RUnlock()
	if ok {
		v.lastSeen = time.Now()
		return v.limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock.
	if v, ok := rl.visitors[ip]; ok {
		v.lastSeen = time.Now()
		return v.limiter
	}

	l := rate.NewLimiter(limit, burst)
	rl.visitors[ip] = &visitor{limiter: l, lastSeen: time.Now()}
	return l
}

func (rl *RateLimiter) sweep() {
	ticker := time.NewTicker(rl.sweepInterval)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.idleTTL {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// realIP extracts the client IP from the request. If trusted CIDRs are
// configured and the direct remote address matches one, it reads the
// X-Forwarded-For or X-Real-IP header (checked in that order). Otherwise it
// returns the host part of RemoteAddr.
func realIP(r *http.Request, trusted []*net.IPNet) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if len(trusted) > 0 {
		remoteIP := net.ParseIP(remoteHost)
		if remoteIP != nil {
			isTrusted := false
			for _, cidr := range trusted {
				if cidr.Contains(remoteIP) {
					isTrusted = true
					break
				}
			}
			if isTrusted {
				if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
					parts := strings.Split(fwd, ",")
					return strings.TrimSpace(parts[0])
				}
				if rip := r.Header.Get("X-Real-IP"); rip != "" {
					return rip
				}
			}
		}
	}

	return remoteHost
}
