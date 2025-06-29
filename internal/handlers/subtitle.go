package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/services"
	"ipmanlk/bettercopelk/internal/sse"
	"net/http"
	"path/filepath"
	"strings"
)

type SubtitleHandler struct {
	service *services.SubtitleService
}

func NewSubtitleHandler(service *services.SubtitleService) *SubtitleHandler {
	return &SubtitleHandler{
		service: service,
	}
}

func splitSources(sourcesStr string) []string {
	if sourcesStr == "" {
		return nil
	}

	sources := strings.Split(sourcesStr, ",")
	for i, src := range sources {
		sources[i] = strings.TrimSpace(src)
	}

	var filteredSources []string
	for _, src := range sources {
		if src != "" {
			filteredSources = append(filteredSources, src)
		}
	}

	return filteredSources
}

func (h *SubtitleHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	var sources []string
	sourcesParam := r.URL.Query().Get("sources")
	if sourcesParam != "" {
		sources = splitSources(sourcesParam)
	}

	req := models.SearchRequest{
		Query:   query,
		Sources: sources,
	}

	if err := h.service.ValidateSources(sources); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.service.Search(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *SubtitleHandler) GetAvailableSources(w http.ResponseWriter, r *http.Request) {
	sources := h.service.GetAvailableSources()

	response := &models.SourcesResponse{
		Sources: sources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *SubtitleHandler) Download(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	source := r.URL.Query().Get("source")

	if url == "" || source == "" {
		http.Error(w, "URL and source parameters are required", http.StatusBadRequest)
		return
	}

	if err := h.service.ValidateSources([]string{source}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := models.DownloadRequest{
		URL:    url,
		Source: source,
	}

	content, filename, err := h.service.Download(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := getContentTypeFromFilename(filename)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Write(content)
}

func (h *SubtitleHandler) SearchStream(w http.ResponseWriter, r *http.Request) {
	req, err := h.parseSearchRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sseWriter, err := sse.NewWriter(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.streamSearchResults(r.Context(), req, sseWriter)
}

func (h *SubtitleHandler) parseSearchRequest(r *http.Request) (models.SearchRequest, error) {
	query := r.URL.Query().Get("query")
	if query == "" {
		return models.SearchRequest{}, fmt.Errorf("query parameter is required")
	}

	var sources []string
	if sourcesParam := r.URL.Query().Get("sources"); sourcesParam != "" {
		sources = splitSources(sourcesParam)
	}

	if err := h.service.ValidateSources(sources); err != nil {
		return models.SearchRequest{}, err
	}

	return models.SearchRequest{
		Query:   query,
		Sources: sources,
	}, nil
}

func (h *SubtitleHandler) streamSearchResults(ctx context.Context, req models.SearchRequest, writer *sse.Writer) {
	resultChan := make(chan models.SearchResult, 10)
	sourceCompleteChan := make(chan models.SourceCompleteEvent, 10)

	go h.service.StreamSearch(ctx, req, resultChan, sourceCompleteChan)

	// Multiplex between result and source completion channels until both are closed
	var resultsDone, sourcesDone bool

	for !resultsDone || !sourcesDone {
		select {
		case <-ctx.Done():
			return

		case result, ok := <-resultChan:
			if !ok {
				resultsDone = true
				continue
			}

			if err := writer.WriteEvent("result", result); err != nil {
				return
			}

		case sourceEvent, ok := <-sourceCompleteChan:
			if !ok {
				sourcesDone = true
				continue
			}

			if err := writer.WriteEvent("source-complete", sourceEvent); err != nil {
				return
			}
		}
	}

	writer.WriteEvent("end", map[string]interface{}{})
}

func getContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".zip":
		return "application/zip"
	case ".srt":
		return "text/plain; charset=utf-8"
	case ".ass", ".ssa":
		return "text/plain; charset=utf-8"
	case ".vtt":
		return "text/vtt; charset=utf-8"
	case ".sub":
		return "text/plain; charset=utf-8"
	default:
		return "application/zip"
	}
}

func (h *SubtitleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/search", h.Search)
	mux.HandleFunc("GET /api/v1/search/stream", h.SearchStream)
	mux.HandleFunc("GET /api/v1/download", h.Download)
	mux.HandleFunc("GET /api/v1/sources", h.GetAvailableSources)
}
