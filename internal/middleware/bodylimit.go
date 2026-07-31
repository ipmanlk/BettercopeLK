package middleware

import (
	"net/http"
)

// BodyLimit limits the request body size for state-changing methods (POST,
// PUT, PATCH) using http.MaxBytesReader. Requests that exceed the limit get an
// early 413 response without reading the full body.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
