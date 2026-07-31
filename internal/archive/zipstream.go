// Package archive provides streaming zip creation and safe extraction of
// subtitle files. It applies bomb-prevention limits configurable through
// the application-wide config.
package archive

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/models"
)

// StreamWriter wraps archive/zip.Writer with byte-limit tracking, filename
// deduplication, and manifest support. It is safe for concurrent use.
type StreamWriter struct {
	mu         sync.Mutex
	zw         *zip.Writer
	totalBytes int64
	maxBytes   int64
	usedNames  map[string]bool
}

// NewWriter creates a StreamWriter that writes to the given io.Writer.
func NewWriter(w io.Writer, cfg *config.Config) *StreamWriter {
	return &StreamWriter{
		zw:        zip.NewWriter(w),
		maxBytes:  cfg.MaxBulkTotalBytes,
		usedNames: make(map[string]bool),
	}
}

// AddFile copies data from r into the zip archive under the given name. If the
// name has already been used in this archive, a numeric suffix is appended.
// It returns an error if adding the entry would exceed the configured total
// byte limit.
func (sw *StreamWriter) AddFile(name string, r io.Reader) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	name = sw.uniqueNameLocked(name)

	h := &zip.FileHeader{
		Name:   name,
		Method: zip.Deflate,
	}
	w, err := sw.zw.CreateHeader(h)
	if err != nil {
		return fmt.Errorf("creating zip entry %q: %w", name, err)
	}

	n, err := io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("writing zip entry %q: %w", name, err)
	}

	sw.totalBytes += n
	if sw.maxBytes > 0 && sw.totalBytes > sw.maxBytes {
		return fmt.Errorf("total zip size %d exceeds limit %d", sw.totalBytes, sw.maxBytes)
	}
	return nil
}

// AddManifest marshals the manifest as pretty-printed JSON and adds it as the
// last entry in the zip archive.
func (sw *StreamWriter) AddManifest(m *models.Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	return sw.AddFile("manifest.json", strings.NewReader(string(data)))
}

// Close finalizes the zip archive. Callers should defer Close after a
// successful NewWriter.
func (sw *StreamWriter) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.zw.Close()
}

// TotalBytes returns the number of bytes written so far.
func (sw *StreamWriter) TotalBytes() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.totalBytes
}

// uniqueNameLocked returns name if it hasn't been used yet, otherwise appends a
// numeric suffix to avoid collision. Must be called with sw.mu held.
func (sw *StreamWriter) uniqueNameLocked(name string) string {
	if !sw.usedNames[name] {
		sw.usedNames[name] = true
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if !sw.usedNames[candidate] {
			sw.usedNames[candidate] = true
			return candidate
		}
	}
}
