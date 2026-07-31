package archive

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func buildZip(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating entry %q: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("writing entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip: %v", err)
	}
	return &buf
}

func TestExtractAndCopyNormal(t *testing.T) {
	cfg := newTestConfig()
	src := buildZip(t, map[string]string{
		"movie.srt": "1\n00:00:01,000 --> 00:00:02,000\nHello\n",
		"movie.ass": "Script Info\n",
	})

	var outBuf bytes.Buffer
	sw := NewWriter(&outBuf, cfg)
	files, totalBytes, err := ExtractAndCopy(cfg, sw, bytes.NewReader(src.Bytes()), int64(src.Len()), "")
	if err != nil {
		t.Fatalf("ExtractAndCopy: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	if totalBytes <= 0 {
		t.Errorf("totalBytes should be > 0, got %d", totalBytes)
	}

	// Verify the output zip is valid.
	zr, err := zip.NewReader(bytes.NewReader(outBuf.Bytes()), int64(outBuf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader on output: %v", err)
	}
	if len(zr.File) != 2 {
		t.Errorf("expected 2 files in output zip, got %d", len(zr.File))
	}
}

func TestExtractAndCopyNestedZip(t *testing.T) {
	cfg := newTestConfig()

	// Build inner zip.
	inner := buildZip(t, map[string]string{
		"inner.srt": "1\n00:00:01,000 --> 00:00:02,000\nNested\n",
	})

	// Wrap inner zip inside outer zip.
	var outerBuf bytes.Buffer
	zw := zip.NewWriter(&outerBuf)
	w, err := zw.Create("subtitles.zip")
	if err != nil {
		t.Fatalf("creating outer entry: %v", err)
	}
	if _, err := io.Copy(w, inner); err != nil {
		t.Fatalf("writing outer entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("closing outer zip: %v", err)
	}

	var outBuf bytes.Buffer
	sw := NewWriter(&outBuf, cfg)
	files, _, err := ExtractAndCopy(cfg, sw, bytes.NewReader(outerBuf.Bytes()), int64(outerBuf.Len()), "")
	if err != nil {
		t.Fatalf("ExtractAndCopy: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file from nested zip, got %d: %v", len(files), files)
	}

	if files[0] != "subtitles/inner.srt" {
		t.Errorf("expected file name 'subtitles/inner.srt', got %q", files[0])
	}
}

func TestExtractAndCopyBombOversizedEntry(t *testing.T) {
	// cfg.MaxFileBytes is 10, but the content written is 13 bytes,
	// so UncompressedSize64 > MaxFileBytes triggers a bomb rejection.
	cfg := newTestConfig()
	cfg.MaxFileBytes = 10 // tiny

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("big.srt")
	io.WriteString(w, "small content")
	zw.Close()

	// Re-read and check.
	sw := NewWriter(io.Discard, cfg)
	_, _, err := ExtractAndCopy(cfg, sw, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "")
	if err == nil {
		t.Fatal("expected error for oversized entry, got nil")
	}
}

func TestExtractAndCopyBombTooManyEntries(t *testing.T) {
	cfg := newTestConfig()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := 0; i < maxEntriesPerSourceZip+1; i++ {
		w, err := zw.Create(string(rune('a'+i)) + ".srt")
		if err != nil {
			t.Fatalf("creating entry: %v", err)
		}
		io.WriteString(w, "x")
	}
	zw.Close()

	sw := NewWriter(io.Discard, cfg)
	_, _, err := ExtractAndCopy(cfg, sw, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "")
	if err == nil {
		t.Fatal("expected error for too many entries, got nil")
	}
}

func TestExtractAndCopyBombCompressionRatio(t *testing.T) {
	cfg := newTestConfig()
	cfg.MaxFileBytes = 1024 * 1024

	// Build a zip where the ratio check fires.
	// We need an entry where UncompressedSize64 / CompressedSize64 > 100.
	// The simplest way: a highly compressible entry (CompressedSize64 very small).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Write ten identical lines that compress extremely well.
	h := &zip.FileHeader{
		Name:   "bomb.srt",
		Method: zip.Deflate,
	}
	w, _ := zw.CreateHeader(h)
	for i := 0; i < 1000; i++ {
		io.WriteString(w, strings.Repeat("A", 100)+"\n")
	}
	zw.Close()

	// Now check if the ratio exceeds the limit. If it doesn't (because the
	// compressed size is still sizable), the test is not useful — skip.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if len(zr.File) == 0 {
		t.Fatal("no entries in test zip")
	}
	f := zr.File[0]

	ratio := float64(f.UncompressedSize64) / float64(f.CompressedSize64)
	t.Logf("ratio: %.1f (uncompressed=%d, compressed=%d)", ratio, f.UncompressedSize64, f.CompressedSize64)

	if ratio <= maxCompressionRatio {
		t.Skipf("ratio %.1f does not exceed max %.0f — test not applicable", ratio, maxCompressionRatio)
	}

	sw := NewWriter(io.Discard, cfg)
	_, _, err = ExtractAndCopy(cfg, sw, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "")
	if err == nil {
		t.Fatal("expected error for high compression ratio, got nil")
	}
}

func TestExtractAndCopyNameDedupeAcrossFlatten(t *testing.T) {
	cfg := newTestConfig()

	// Build inner zip with "movie.srt".
	inner := buildZip(t, map[string]string{
		"movie.srt": "1\n00:00:01\nFirst\n",
	})

	// Build outer zip with the inner zip AND a separate movie.srt.
	var outerBuf bytes.Buffer
	zw := zip.NewWriter(&outerBuf)

	w, _ := zw.Create("subtitles.zip")
	io.Copy(w, inner)

	w, _ = zw.Create("movie.srt")
	io.WriteString(w, "1\n00:00:01\nSecond\n")
	zw.Close()

	var outBuf bytes.Buffer
	sw := NewWriter(&outBuf, cfg)
	files, _, err := ExtractAndCopy(cfg, sw, bytes.NewReader(outerBuf.Bytes()), int64(outerBuf.Len()), "")
	if err != nil {
		t.Fatalf("ExtractAndCopy: %v", err)
	}
	sw.Close()

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	// The nested zip's entry gets prefix "subtitles/", so it should be
	// "subtitles/movie.srt". The flat entry stays "movie.srt". No collision.
	hasNested := false
	hasFlat := false
	for _, f := range files {
		if f == "subtitles/movie.srt" {
			hasNested = true
		}
		if f == "movie.srt" {
			hasFlat = true
		}
	}
	if !hasNested {
		t.Errorf("expected nested file 'subtitles/movie.srt', got %v", files)
	}
	if !hasFlat {
		t.Errorf("expected flat file 'movie.srt', got %v", files)
	}
}
