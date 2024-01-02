package search

import (
	"ipmanlk/bettercopelk/models"
	"testing"
	"time"
)

func TestSearchSites(t *testing.T) {
	query := "life"
	resultsChan := make(chan []models.SearchResult, 10) // Buffered channel

	go SearchSources(query, resultsChan)

	// Timeout to ensure the test doesn't run indefinitely
	timeout := time.After(30 * time.Second)

	select {
	case <-timeout:
		t.Fatal("Test timed out. SearchSites took too long to return results.")
	case results := <-resultsChan:
		if len(results) == 0 {
			t.Error("SearchSites returned no results.")
		}
	}
}

func TestSearchSite(t *testing.T) {
	query := "life"

	for source := range sourceConfigs {
		results, err := SearchSource(source, query)
		if err != nil {
			t.Errorf("Error searching source %v: %v", source, err)
			continue
		}

		if len(results) == 0 {
			t.Errorf("No results returned for source: %v", source)
		}
	}
}
