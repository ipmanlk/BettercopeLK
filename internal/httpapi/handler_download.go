package httpapi

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"ipmanlk/bettercopelk/internal/middleware"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/token"
)

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	sourceParam := r.URL.Query().Get("source")

	if pathParam == "" || sourceParam == "" {
		writeError(w, r, http.StatusBadRequest, "path and source are required")
		return
	}

	// Validate source exists.
	if s.registry.Get(sourceParam) == nil {
		writeError(w, r, http.StatusBadRequest, "unknown source: "+sourceParam)
		return
	}

	// Validate the path token decodes before calling the service.
	if _, err := token.DecodePath(pathParam); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid path: "+err.Error())
		return
	}

	req := models.DownloadRequest{
		Path:   pathParam,
		Source: sourceParam,
	}

	content, filename, err := s.service.Download(r.Context(), req)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "download failed: "+err.Error())
		return
	}

	contentType := detectContentType(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleBulkDownload(w http.ResponseWriter, r *http.Request) {
	var req models.BulkDownloadRequest

	// Body size is already limited by the BodyLimit middleware.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if len(req.Items) == 0 {
		writeError(w, r, http.StatusBadRequest, "items is required")
		return
	}
	if len(req.Items) > s.cfg.MaxBulkItems {
		writeError(w, r, http.StatusBadRequest, fmt.Sprintf("items exceeds max %d", s.cfg.MaxBulkItems))
		return
	}

	// Validate each item's source and path token before streaming starts, so
	// a bad request gets a 400 instead of a truncated zip.
	for _, item := range req.Items {
		if s.registry.Get(item.Source) == nil {
			writeError(w, r, http.StatusBadRequest, fmt.Sprintf("unknown source %q", item.Source))
			return
		}
		if _, err := token.DecodePath(item.Path); err != nil {
			writeError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid path for item: %v", err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="subtitles.zip"`)
	w.WriteHeader(http.StatusOK)

	manifest, err := s.service.DownloadBulk(r.Context(), req, w)
	if err != nil {
		// If writing the archive fails mid-stream, the response is already
		// partially sent. We log the error and return; the client gets a
		// truncated zip with the manifest entries written so far.
		s.logger.Error("bulk download failed mid-stream",
			"error", err,
			"request_id", middleware.GetRequestID(r.Context()),
		)
		return
	}

	_ = manifest
}

// detectContentType returns the Content-Type for a subtitle file based on its
// extension.
func detectContentType(filename string) string {
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
		// Fallback to MIME type detection.
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		return "application/octet-stream"
	}
}
