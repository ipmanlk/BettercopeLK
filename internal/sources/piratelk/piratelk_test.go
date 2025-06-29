package piratelk

import (
	"context"
	"ipmanlk/bettercopelk/internal/models"
	"testing"
	"time"
)

func TestPirateLK_Name(t *testing.T) {
	source := New()
	expected := "piratelk"
	if got := source.Name(); got != expected {
		t.Errorf("Name() = %v, want %v", got, expected)
	}
}

func TestPirateLK_IsAvailable(t *testing.T) {
	source := New()

	available := source.IsAvailable()
	t.Logf("PirateLK availability: %v", available)
}

func TestPirateLK_Search(t *testing.T) {
	source := New()

	if !source.IsAvailable() {
		t.Skip("PirateLK is not available, skipping search test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := models.SearchRequest{
		Query: "batman",
	}

	results, err := source.Search(ctx, req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one search result, got none")
		return
	}

	result := results[0]
	if result.Title == "" {
		t.Error("First result title is empty")
	}
	if result.URL == "" {
		t.Error("First result URL is empty")
	}
	if result.Source != "piratelk" {
		t.Errorf("First result source = %v, want %v", result.Source, "piratelk")
	}

	t.Logf("Found %d results for 'batman'", len(results))
	t.Logf("First result: Title=%s, URL=%s", result.Title, result.URL)
}

func TestPirateLK_Download(t *testing.T) {
	source := New()

	if !source.IsAvailable() {
		t.Skip("PirateLK is not available, skipping download test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	searchReq := models.SearchRequest{
		Query: "batman",
	}

	results, err := source.Search(ctx, searchReq)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No search results found for batman")
	}

	downloadURL := results[0].URL
	t.Logf("Testing download from URL: %s", downloadURL)

	content, filename, err := source.Download(ctx, downloadURL)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("Downloaded content is empty")
	}

	if filename == "" {
		t.Error("Filename is empty")
	}

	t.Logf("Downloaded file: %s (%d bytes)", filename, len(content))

	if len(content) < 100 {
		t.Errorf("Downloaded content seems too small: %d bytes", len(content))
	}
}
