// Package models defines the domain types shared across the application.
package models

import "time"

// SearchRequest represents a subtitle search request.
type SearchRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources,omitempty"`
}

// SearchResult is a single subtitle hit from a source. Path is the encoded
// relative URL token (see internal/token); the full URL never leaves the
// server.
type SearchResult struct {
	Title  string `json:"title"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

// SearchResponse wraps a list of search results.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// DownloadRequest is a request to download a single subtitle. Path is the
// encoded relative URL token from a SearchResult.
type DownloadRequest struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

// BulkDownloadRequest is a request to download multiple subtitles.
type BulkDownloadRequest struct {
	Items []BulkItem `json:"items"`
}

// BulkItem is one entry in a bulk download request. Path is the encoded
// relative URL token from a SearchResult.
type BulkItem struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

// SourcesResponse is the response for the GET /sources endpoint.
type SourcesResponse struct {
	Sources []SourceInfo `json:"sources"`
}

// SourceInfo describes one subtitle source.
type SourceInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// SubtitleFile holds the raw content of a downloaded subtitle.
type SubtitleFile struct {
	Filename string `json:"filename"`
	Content  []byte `json:"-"`
}

// SSE event types for the search stream endpoint.

// ResultEvent is emitted for each subtitle hit during a streamed search.
type ResultEvent struct {
	Title  string `json:"title"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

// SourceCompleteEvent is emitted when a source finishes its search.
type SourceCompleteEvent struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// ErrorEvent is emitted when a non-fatal error occurs during a streamed search.
type ErrorEvent struct {
	Message string `json:"message"`
}

// DoneEvent marks the end of a search stream.
type DoneEvent struct {
	Total int `json:"total,omitempty"`
}

// ErrorEnvelope is the standard JSON error response body.
type ErrorEnvelope struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// Manifest describes the result of a bulk download, written as the first entry
// in the output zip.
type Manifest struct {
	Items     []ManifestItem `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
}

// ManifestItem records the outcome of one bulk download item. Path is the
// encoded relative URL token from the request, so the full URL stays
// server-side.
type ManifestItem struct {
	Path   string   `json:"path"`
	Source string   `json:"source"`
	OK     bool     `json:"ok"`
	Files  []string `json:"files,omitempty"`
	Bytes  int64    `json:"bytes,omitempty"`
	Error  string   `json:"error,omitempty"`
}
