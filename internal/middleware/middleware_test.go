package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ipmanlk/bettercopelk/internal/models"
)

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`ok`))
}

func panickingHandler(_ http.ResponseWriter, _ *http.Request) {
	panic("test panic")
}

// --- RequestID ---

func TestRequestID_PropagatesHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RequestID(http.HandlerFunc(okHandler)).ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected non-empty X-Request-ID header")
	}
}

func TestRequestID_RespectsIncoming(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "incoming-id")
	rec := httptest.NewRecorder()

	RequestID(http.HandlerFunc(okHandler)).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "incoming-id" {
		t.Fatalf("expected %q, got %q", "incoming-id", got)
	}
}

func TestRequestID_ContextValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	var got string
	RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if got == "" {
		t.Fatal("expected non-empty request ID in context")
	}
	if got != rec.Header().Get("X-Request-ID") {
		t.Fatal("context ID and response header must match")
	}
}

func TestGetRequestID_Missing(t *testing.T) {
	if v := GetRequestID(context.Background()); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
}

// --- SecurityHeaders ---

func TestSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	SecurityHeaders(http.HandlerFunc(okHandler)).ServeHTTP(rec, req)

	tests := []struct {
		key, expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "no-referrer"},
		{"X-Frame-Options", "DENY"},
		{"Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"},
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "GET, POST, OPTIONS"},
	}
	for _, tt := range tests {
		v := rec.Header().Get(tt.key)
		if v != tt.expected {
			t.Errorf("header %q: expected %q, got %q", tt.key, tt.expected, v)
		}
	}
}

func TestSecurityHeaders_Options(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()

	// An OPTIONS request should return 200 without calling the next handler.
	var called bool
	SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if called {
		t.Fatal("OPTIONS request should not reach the next handler")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// --- Recovery ---

func TestRecovery_Returns500JSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RequestID(
		Recovery(logger)(http.HandlerFunc(panickingHandler)),
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var env models.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decoding error envelope: %v", err)
	}
	if env.Error != "internal server error" {
		t.Fatalf("expected 'internal server error', got %q", env.Error)
	}
	if env.RequestID == "" {
		t.Fatal("expected non-empty request_id in error envelope")
	}

	// Verify the panic was logged.
	if !strings.Contains(buf.String(), "handler panic") {
		t.Fatal("expected 'handler panic' in log output")
	}
}

// --- BodyLimit ---

func TestBodyLimit_RejectsLargeBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("too large body"))
	rec := httptest.NewRecorder()

	var readErr error
	BodyLimit(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if readErr == nil {
		t.Fatal("expected error from MaxBytesReader when body exceeds limit")
	}
	if _, ok := readErr.(*http.MaxBytesError); !ok {
		t.Fatalf("expected *http.MaxBytesError, got %T: %v", readErr, readErr)
	}
	// The middleware wraps the body correctly; the handler is responsible for
	// checking the error. MaxBytesReader in Go 1.26 does not automatically write
	// the 413 status — that is left to the handler.
}

func TestBodyLimit_AcceptsSmallBody(t *testing.T) {
	body := "small"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	BodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBodyLimit_DoesNotAffectGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?large=true&query=huge", nil)
	rec := httptest.NewRecorder()

	BodyLimit(10)(http.HandlerFunc(okHandler)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// --- RateLimit ---

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(120, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rl.SetTrustedCIDRs(nil)

	middleware := rl.Middleware(600, 100) // generous route limit
	handler := middleware(http.HandlerFunc(okHandler))

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 within limit, got %d", rec.Code)
		}
	}
}

func TestRateLimit_RejectsExcess(t *testing.T) {
	rl := NewRateLimiter(120, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Tight limit: 1 rpm with a burst of 1.
	middleware := rl.Middleware(1, 1)
	handler := middleware(http.HandlerFunc(okHandler))

	// First request should pass.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request should pass: got %d", rec1.Code)
	}

	// Second request should be rate-limited.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	rl := NewRateLimiter(120, slog.New(slog.NewTextHandler(io.Discard, nil)))
	middleware := rl.Middleware(1, 1)
	handler := middleware(http.HandlerFunc(okHandler))

	// Exhaust the limiter.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Next request gets 429.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if ra := rec2.Header().Get("Retry-After"); ra == "" {
		t.Fatal("expected Retry-After header")
	}
}

// --- AccessLog ---

func TestAccessLog_OutputShape(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	RequestID(
		AccessLog(logger)(http.HandlerFunc(okHandler)),
	).ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "request") {
		t.Fatal("log output should contain 'request' message")
	}
	if !strings.Contains(output, "GET") {
		t.Fatal("log output should contain HTTP method")
	}
	if !strings.Contains(output, "/test-path") {
		t.Fatal("log output should contain the request path")
	}
	if !strings.Contains(output, "200") {
		t.Fatal("log output should contain status code")
	}
}
