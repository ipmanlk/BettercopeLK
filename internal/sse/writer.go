package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Writer handles writing Server-Sent Events to the HTTP response
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &Writer{
		w:       w,
		flusher: flusher,
	}, nil
}

func (s *Writer) WriteEvent(event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	_, err = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, jsonData)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	s.flusher.Flush()
	return nil
}

func (s *Writer) WriteError(err error) {
	fmt.Fprintf(s.w, "event: error\ndata: %s\n\n", err.Error())
	s.flusher.Flush()
}
