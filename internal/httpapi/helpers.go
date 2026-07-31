package httpapi

import (
	"encoding/json"
	"net/http"

	"ipmanlk/bettercopelk/internal/middleware"
)

// writeJSON serializes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error envelope with the given status and message.
// It extracts the request ID from the request context.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	reqID := middleware.GetRequestID(r.Context())
	writeJSON(w, status, map[string]any{
		"error":      msg,
		"request_id": reqID,
	})
}
