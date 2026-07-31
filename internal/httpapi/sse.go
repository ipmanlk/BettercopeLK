// Package httpapi provides typed SSE event writing.
package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Writer handles Server-Sent Events. It sets the required headers on
// construction and provides WriteEvent for emitting typed events.
type Writer struct {
	w  http.ResponseWriter
	fl http.Flusher
}

// NewWriter creates an SSE writer and sets the required headers. It returns an
// error if the ResponseWriter does not support flushing (http.Flusher).
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	fl, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	return &Writer{w: w, fl: fl}, nil
}

// WriteEvent marshals data as JSON and writes an SSE event with the given
// event type. It flushes after each write so the client receives partial
// results as they arrive.
func (sw *Writer) WriteEvent(event string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal SSE event %q: %w", event, err)
	}
	_, err = fmt.Fprintf(sw.w, "event: %s\ndata: %s\n\n", event, string(b))
	if err != nil {
		return err
	}
	sw.fl.Flush()
	return nil
}
