package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter and records the status code and
// bytes written so middleware downstream can inspect them.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(p)
	rw.bytes += int64(n)
	return n, err
}

// Flush implements http.Flusher by forwarding to the underlying ResponseWriter
// if it supports flushing. This is required for Server-Sent Events to work
// through the middleware stack.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AccessLog returns middleware that logs every request with structured fields.
// The log level is INFO for 2xx/3xx, WARN for 4xx, and ERROR for 5xx.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := wrapResponseWriter(w)
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			level := slog.LevelInfo
			if rw.status >= 500 {
				level = slog.LevelError
			} else if rw.status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", duration),
				slog.Int64("bytes", rw.bytes),
				slog.String("request_id", GetRequestID(r.Context())),
			)
		})
	}
}
