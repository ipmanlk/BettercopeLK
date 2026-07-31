package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"ipmanlk/bettercopelk/internal/models"
)

// Recovery catches panics in downstream handlers, logs the stack trace, and
// returns a 500 response with a JSON error envelope.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					reqID := GetRequestID(r.Context())
					logger.Error("handler panic",
						"panic", fmt.Sprintf("%v", rec),
						"stack", string(debug.Stack()),
						"path", r.URL.Path,
						"request_id", reqID,
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(models.ErrorEnvelope{
						Error:     "internal server error",
						RequestID: reqID,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
