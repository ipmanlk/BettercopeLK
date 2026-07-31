// Package archive provides streaming zip creation and safe extraction of
// subtitle files.
package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"

	"ipmanlk/bettercopelk/internal/config"
)

// maxEntriesPerSourceZip is the maximum number of files allowed inside a single
// source subtitle zip. This guards against zip-bomb attacks that use thousands
// of tiny files.
const maxEntriesPerSourceZip = 100

// maxCompressionRatio is the maximum ratio of uncompressed size to compressed
// size permitted. A ratio above this threshold is treated as a bomb.
const maxCompressionRatio = 100.0

// ExtractAndCopy reads a source zip from src, validates each entry against the
// configured limits, and writes safe entries into out. It returns the list of
// filenames written and the total uncompressed bytes.
//
// Nested zips (entries whose name ends in .zip) are flattened one level only.
// The prefix is prepended to every entry name written to the output archive.
func ExtractAndCopy(cfg *config.Config, out *StreamWriter, src io.ReaderAt, size int64, prefix string) ([]string, int64, error) {
	return extractAndCopy(cfg, out, src, size, prefix, 0)
}

// extractAndCopy is the internal implementation with depth tracking.
func extractAndCopy(cfg *config.Config, out *StreamWriter, src io.ReaderAt, size int64, prefix string, depth int) ([]string, int64, error) {
	zr, err := zip.NewReader(src, size)
	if err != nil {
		return nil, 0, fmt.Errorf("opening source zip: %w", err)
	}

	if len(zr.File) > maxEntriesPerSourceZip {
		return nil, 0, fmt.Errorf("source zip has %d entries, max %d", len(zr.File), maxEntriesPerSourceZip)
	}

	var written []string
	var totalBytes int64

	for _, f := range zr.File {
		// If this entry is itself a zip, flatten one level (depth 0 only).
		if IsZipFile(f.Name) {
			if depth >= 1 {
				return nil, 0, fmt.Errorf("nested zip %q exceeds max depth of 1", f.Name)
			}
			// Extract the nested zip into memory and process at depth+1.
			nestedName := strings.TrimSuffix(f.Name, ".zip")
			nestedPrefix := nestedName
			if prefix != "" {
				nestedPrefix = filepath.Join(prefix, nestedName)
			}

			rc, err := f.Open()
			if err != nil {
				return nil, 0, fmt.Errorf("opening nested zip entry %q: %w", f.Name, err)
			}

			nestedData, err := io.ReadAll(io.LimitReader(rc, cfg.MaxFileBytes))
			rc.Close()
			if err != nil {
				return nil, 0, fmt.Errorf("reading nested zip entry %q: %w", f.Name, err)
			}

			if int64(len(nestedData)) > cfg.MaxFileBytes {
				return nil, 0, fmt.Errorf("nested zip %q exceeds MaxFileBytes", f.Name)
			}

			nestedWritten, nestedBytes, err := extractAndCopy(cfg, out, bytes.NewReader(nestedData), int64(len(nestedData)), nestedPrefix, depth+1)
			if err != nil {
				return nil, 0, fmt.Errorf("extracting nested zip %q: %w", f.Name, err)
			}
			written = append(written, nestedWritten...)
			totalBytes += nestedBytes
			continue
		}

		// Bomb checks.
		if err := checkBomb(cfg, f); err != nil {
			return nil, 0, fmt.Errorf("entry %q: %w", f.Name, err)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, 0, fmt.Errorf("opening entry %q: %w", f.Name, err)
		}

		entryName := f.Name
		if prefix != "" {
			entryName = filepath.Join(prefix, f.Name)
		}

		if err := out.AddFile(entryName, io.LimitReader(rc, cfg.MaxFileBytes)); err != nil {
			rc.Close()
			return nil, 0, fmt.Errorf("adding entry %q to output: %w", entryName, err)
		}
		rc.Close()

		written = append(written, entryName)
		totalBytes += int64(f.UncompressedSize64)
	}

	return written, totalBytes, nil
}

// IsZipFile returns true if the filename has a .zip extension (case-insensitive).
func IsZipFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".zip")
}

// checkBomb validates a zip file entry against bomb patterns: oversized
// uncompressed data, too-high compression ratio, and other suspicious
// characteristics.
func checkBomb(cfg *config.Config, f *zip.File) error {
	if f.UncompressedSize64 > uint64(cfg.MaxFileBytes) {
		return fmt.Errorf("uncompressed size %d exceeds limit %d", f.UncompressedSize64, cfg.MaxFileBytes)
	}

	// Avoid division by zero for zero-compressed-size entries.
	compressed := f.CompressedSize64
	if compressed == 0 {
		compressed = 1
	}

	ratio := float64(f.UncompressedSize64) / float64(compressed)
	if ratio > maxCompressionRatio && !math.IsInf(ratio, 0) {
		return fmt.Errorf("compression ratio %.1f exceeds max %.0f", ratio, maxCompressionRatio)
	}

	return nil
}
