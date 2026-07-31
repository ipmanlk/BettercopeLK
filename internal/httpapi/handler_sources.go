package httpapi

import (
	"net/http"

	"ipmanlk/bettercopelk/internal/models"
)

// handleSources returns a list of all registered sources with their
// availability status. Availability checks are cached by the registry.
func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	infos := s.registry.List(r.Context())
	resp := models.SourcesResponse{Sources: make([]models.SourceInfo, 0, len(infos))}
	for _, info := range infos {
		resp.Sources = append(resp.Sources, models.SourceInfo{
			ID:        info.ID,
			Name:      info.Name,
			Available: info.Available,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
