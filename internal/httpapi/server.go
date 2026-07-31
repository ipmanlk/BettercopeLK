// Package httpapi provides the HTTP routing layer and server lifecycle.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/httpclient"
	"ipmanlk/bettercopelk/internal/middleware"
	"ipmanlk/bettercopelk/internal/security"
	"ipmanlk/bettercopelk/internal/source"
	"ipmanlk/bettercopelk/internal/subtitle"
	"ipmanlk/bettercopelk/internal/web"
)

// Server holds all dependencies and provides the HTTP server lifecycle.
type Server struct {
	cfg      *config.Config
	logger   *slog.Logger
	registry *source.Registry
	guard    *security.Guard
	service  *subtitle.Service
	client   *httpclient.Client
	limiter  *middleware.RateLimiter
	srv      *http.Server
}

// New creates a Server with all required dependencies wired.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	registry *source.Registry,
	guard *security.Guard,
	service *subtitle.Service,
	client *httpclient.Client,
) *Server {
	rl := middleware.NewRateLimiter(cfg.RateGlobalRPM, logger)
	rl.SetTrustedCIDRs(cfg.TrustedProxies)

	return &Server{
		cfg:      cfg,
		logger:   logger,
		registry: registry,
		guard:    guard,
		service:  service,
		client:   client,
		limiter:  rl,
	}
}

// RegisterRoutes registers all routes on the given mux. Rate-limited routes
// are wrapped with per-route rate limiters. Static assets are served from the
// embedded web package.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// API v1 routes with per-route rate limiting.
	// Sources endpoint uses the search rate limit (lightweight, called once per page load).
	mux.Handle("GET /api/v1/sources", s.limiter.Middleware(s.cfg.RateSearchRPM, 10)(http.HandlerFunc(s.handleSources)))

	// Search endpoints.
	sSearch := s.limiter.Middleware(s.cfg.RateSearchRPM, 10)
	mux.Handle("GET /api/v1/search", sSearch(http.HandlerFunc(s.handleSearch)))
	mux.Handle("GET /api/v1/search/stream", sSearch(http.HandlerFunc(s.handleSearchStream)))

	// Single download.
	sDownload := s.limiter.Middleware(s.cfg.RateDownloadRPM, 5)
	mux.Handle("GET /api/v1/download", sDownload(http.HandlerFunc(s.handleDownload)))

	// Bulk download (more expensive, lower limit).
	sBulk := s.limiter.Middleware(s.cfg.RateBulkRPM, 2)
	mux.Handle("POST /api/v1/download/bulk", sBulk(http.HandlerFunc(s.handleBulkDownload)))

	// Health endpoints (no rate limit — load balancers need them).
	mux.Handle("GET /healthz", http.HandlerFunc(s.handleHealthz))
	mux.Handle("GET /readyz", http.HandlerFunc(s.handleReadyz))

	// Static assets (no rate limit).
	fs := web.FileServer()
	mux.Handle("GET /", fs)
	mux.Handle("GET /index.html", fs)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", fs))
}

// Handler returns the fully-wrapped http.Handler with the global middleware
// stack applied: requestid → recovery → accesslog → security → bodylimit.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	// Apply global middleware stack (innermost to outermost).
	h := http.Handler(mux)
	h = middleware.BodyLimit(s.cfg.MaxBodyBytes)(h)
	h = middleware.SecurityHeaders(h)
	h = middleware.AccessLog(s.logger)(h)
	h = middleware.Recovery(s.logger)(h)
	h = middleware.RequestID(h)
	return h
}

// Start creates an http.Server with the configured timeouts and begins
// listening. It blocks until the server is shut down.
func (s *Server) Start() error {
	addr := ":" + strconv.Itoa(s.cfg.Port)
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	s.logger.Info("server starting", "addr", addr)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server with a timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

// WaitForSignal blocks until SIGINT or SIGTERM is received, then calls
// Shutdown with the configured read-header timeout. It returns the result of
// Shutdown.
func (s *Server) WaitForSignal() error {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	s.logger.Info("shutting down", "signal", sig)
	return s.Shutdown(s.cfg.ReadHeaderTimeout)
}
