package archive

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/models"
)

func newTestConfig() *config.Config {
	return &config.Config{
		MaxFileBytes:      1024 * 1024,      // 1 MB
		MaxBulkTotalBytes: 10 * 1024 * 1024, // 10 MB
	}
}

func TestStreamWriterRoundTrip(t *testing.T) {
	cfg := newTestConfig()
	var buf bytes.Buffer
	sw := NewWriter(&buf, cfg)

	if err := sw.AddFile("hello.txt", strings.NewReader("world")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	if err := sw.AddFile("foo/bar.txt", strings.NewReader("baz")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	if len(zr.File) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(zr.File))
	}

	names := make(map[string]string)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening %q: %v", f.Name, err)
		}
		var content bytes.Buffer
		content.ReadFrom(rc)
		rc.Close()
		names[f.Name] = content.String()
	}

	if names["hello.txt"] != "world" {
		t.Errorf("hello.txt = %q, want %q", names["hello.txt"], "world")
	}
	if names["foo/bar.txt"] != "baz" {
		t.Errorf("foo/bar.txt = %q, want %q", names["foo/bar.txt"], "baz")
	}
}

func TestStreamWriterNameDedupe(t *testing.T) {
	cfg := newTestConfig()
	var buf bytes.Buffer
	sw := NewWriter(&buf, cfg)

	if err := sw.AddFile("dup.srt", strings.NewReader("first")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	if err := sw.AddFile("dup.srt", strings.NewReader("second")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	if err := sw.AddFile("dup.srt", strings.NewReader("third")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	if len(zr.File) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(zr.File))
	}

	got := make(map[string]string)
	for _, f := range zr.File {
		rc, _ := f.Open()
		var c bytes.Buffer
		c.ReadFrom(rc)
		rc.Close()
		got[f.Name] = c.String()
	}

	if got["dup.srt"] != "first" {
		t.Errorf("dup.srt = %q, want %q", got["dup.srt"], "first")
	}
	if got["dup (1).srt"] != "second" {
		t.Errorf("dup (1).srt = %q, want %q", got["dup (1).srt"], "second")
	}
	if got["dup (2).srt"] != "third" {
		t.Errorf("dup (2).srt = %q, want %q", got["dup (2).srt"], "third")
	}
}

func TestStreamWriterAddManifest(t *testing.T) {
	cfg := newTestConfig()
	var buf bytes.Buffer
	sw := NewWriter(&buf, cfg)

	m := &models.Manifest{
		Items: []models.ManifestItem{
			{URL: "http://example.com/movie", Source: "test", OK: true, Files: []string{"movie.srt"}, Bytes: 100},
			{URL: "http://example.com/fail", Source: "test", OK: false, Error: "upstream 502"},
		},
	}
	if err := sw.AddManifest(m); err != nil {
		t.Fatalf("AddManifest: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if len(zr.File) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(zr.File))
	}
	if zr.File[0].Name != "manifest.json" {
		t.Errorf("entry name = %q, want %q", zr.File[0].Name, "manifest.json")
	}

	rc, _ := zr.File[0].Open()
	var content bytes.Buffer
	content.ReadFrom(rc)
	rc.Close()

	if !strings.Contains(content.String(), `"ok": true`) {
		t.Errorf("manifest should contain ok: true")
	}
	if !strings.Contains(content.String(), "upstream 502") {
		t.Errorf("manifest should contain error message")
	}
}

func TestStreamWriterByteLimit(t *testing.T) {
	cfg := &config.Config{
		MaxBulkTotalBytes: 10, // very tight
	}
	var buf bytes.Buffer
	sw := NewWriter(&buf, cfg)

	if err := sw.AddFile("small.txt", strings.NewReader("hi")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	// Second file exceeds the total limit.
	err := sw.AddFile("big.txt", strings.NewReader("this is more than ten bytes"))
	if err == nil {
		t.Fatal("expected error for exceeding byte limit, got nil")
	}
}
