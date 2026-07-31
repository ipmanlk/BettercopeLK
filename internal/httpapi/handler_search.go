package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"ipmanlk/bettercopelk/internal/models"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	req, err := s.parseSearchRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if utf8.RuneCountInString(req.Query) > 128 {
		writeError(w, r, http.StatusBadRequest, "query must be at most 128 characters")
		return
	}

	resp, err := s.service.Search(r.Context(), *req)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSearchStream(w http.ResponseWriter, r *http.Request) {
	req, err := s.parseSearchRequest(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if utf8.RuneCountInString(req.Query) > 128 {
		writeError(w, r, http.StatusBadRequest, "query must be at most 128 characters")
		return
	}

	sw, err := NewWriter(w)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	resultCh := make(chan models.SearchResult, 10)
	completeCh := make(chan models.SourceCompleteEvent, 10)
	go s.service.StreamSearch(r.Context(), *req, resultCh, completeCh)

	var resultsDone, sourcesDone bool
	total := 0
	for !resultsDone || !sourcesDone {
		select {
		case <-r.Context().Done():
			return
		case result, ok := <-resultCh:
			if !ok {
				resultsDone = true
				continue
			}
			total++
			if err := sw.WriteEvent("result", result); err != nil {
				return
			}
		case ev, ok := <-completeCh:
			if !ok {
				sourcesDone = true
				continue
			}
			if err := sw.WriteEvent("source-complete", ev); err != nil {
				return
			}
		}
	}

	_ = sw.WriteEvent("done", models.DoneEvent{Total: total})
}

// parseSearchRequest extracts and validates the query and sources parameters.
func (s *Server) parseSearchRequest(r *http.Request) (*models.SearchRequest, error) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	var sources []string
	if s := r.URL.Query().Get("sources"); s != "" {
		sources = splitSources(s)
	}

	// Validate source IDs.
	for _, id := range sources {
		if s.registry.Get(id) == nil {
			return nil, fmt.Errorf("unknown source: %s", id)
		}
	}

	return &models.SearchRequest{
		Query:   query,
		Sources: sources,
	}, nil
}

// splitSources splits a comma-separated source list and trims whitespace.
func splitSources(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
