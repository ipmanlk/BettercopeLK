package cineru

import (
	"context"
	"ipmanlk/bettercopelk/internal/models"
	"testing"
	"time"
)

func TestCineruLK_Name(t *testing.T) {
	source := New()
	expected := "cineru"
	if got := source.Name(); got != expected {
		t.Errorf("Name() = %v, want %v", got, expected)
	}
}

func TestCineruLK_IsAvailable(t *testing.T) {
	source := New()

	// This test might fail if the website is down, so we'll just check that it returns a boolean
	available := source.IsAvailable()
	t.Logf("CineruLK availability: %v", available)

	// We don't assert true/false because the website might be temporarily unavailable
	// The test passes if it doesn't panic and returns a boolean
}

func TestCineruLK_Search(t *testing.T) {
	source := New()

	// Skip test if source is not available
	if !source.IsAvailable() {
		t.Skip("CineruLK is not available, skipping search test")
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

	// Verify first result structure
	result := results[0]
	if result.Title == "" {
		t.Error("First result title is empty")
	}
	if result.URL == "" {
		t.Error("First result URL is empty")
	}
	if result.Source != "cineru" {
		t.Errorf("First result source = %v, want %v", result.Source, "cineru")
	}

	t.Logf("Found %d results for 'batman'", len(results))
	t.Logf("First result: Title=%s, URL=%s", result.Title, result.URL)
}

func TestCineruLK_Download(t *testing.T) {
	source := New()

	// Skip test if source is not available
	if !source.IsAvailable() {
		t.Skip("CineruLK is not available, skipping download test")
	}

	// First, get search results to have a valid download URL
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

	// Use the first result for download test
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

	// Basic validation that we got some content
	if len(content) < 100 {
		t.Errorf("Downloaded content seems too small: %d bytes", len(content))
	}
}
