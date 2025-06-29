package services

import (
	"context"
	"fmt"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/sources"
	"sync"
)

type SubtitleService struct {
	sourceManager *sources.Manager
}

func NewSubtitleService(sourceManager *sources.Manager) *SubtitleService {
	return &SubtitleService{
		sourceManager: sourceManager,
	}
}

func (s *SubtitleService) Search(ctx context.Context, req models.SearchRequest) (*models.SearchResponse, error) {
	var allResults []models.SearchResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	sourcesToSearch := make(map[string]sources.Source)

	if len(req.Sources) > 0 {
		for _, sourceName := range req.Sources {
			source, exists := s.sourceManager.GetSource(sourceName)
			if !exists {
				continue
			}

			sourcesToSearch[sourceName] = source
		}

		if len(sourcesToSearch) == 0 {
			return nil, fmt.Errorf("none of the requested sources are available")
		}
	} else {
		for name, source := range s.sourceManager.GetAllSources() {
			sourcesToSearch[name] = source
		}
	}

	for name, source := range sourcesToSearch {
		wg.Add(1)
		go func(src sources.Source, srcName string) {
			defer wg.Done()

			results, err := src.Search(ctx, req)
			if err != nil {
				fmt.Printf("Search failed for source %s: %v\n", srcName, err)
				return
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(source, name)
	}

	wg.Wait()

	return &models.SearchResponse{
		Results: allResults,
	}, nil
}

func (s *SubtitleService) Download(ctx context.Context, req models.DownloadRequest) ([]byte, string, error) {

	source, exists := s.sourceManager.GetSource(req.Source)
	if !exists {
		return nil, "", fmt.Errorf("source '%s' not found", req.Source)
	}

	content, filename, err := source.Download(ctx, req.URL)
	if err != nil {
		return nil, "", fmt.Errorf("download failed for source %s: %w", req.Source, err)
	}

	return content, filename, nil
}

func (s *SubtitleService) GetAvailableSources() []string {
	return s.sourceManager.GetAvailableSources()
}
func (s *SubtitleService) ValidateSources(sources []string) error {
	if len(sources) == 0 {
		return nil
	}

	for _, sourceName := range sources {
		_, exists := s.sourceManager.GetSource(sourceName)
		if !exists {
			return fmt.Errorf("invalid source: %s", sourceName)
		}
	}

	return nil
}

func (s *SubtitleService) StreamSearch(ctx context.Context, req models.SearchRequest, resultChan chan<- models.SearchResult, sourceCompleteChan chan<- models.SourceCompleteEvent) {
	wg := sync.WaitGroup{}

	sourcesToSearch := make(map[string]sources.Source)

	if len(req.Sources) > 0 {
		for _, sourceName := range req.Sources {
			source, exists := s.sourceManager.GetSource(sourceName)
			if !exists {
				continue
			}
			sourcesToSearch[sourceName] = source
		}

		if len(sourcesToSearch) == 0 {
			close(resultChan)
			close(sourceCompleteChan)
			return
		}
	} else {
		for name, source := range s.sourceManager.GetAllSources() {
			sourcesToSearch[name] = source
		}
	}

	for name, source := range sourcesToSearch {
		wg.Add(1)
		go func(src sources.Source, srcName string) {
			defer wg.Done()

			results, err := src.Search(ctx, req)
			if err != nil {
				fmt.Printf("Search failed for source %s: %v\n", srcName, err)
				sourceCompleteChan <- models.SourceCompleteEvent{
					Source: srcName,
					Count:  0,
				}
				return
			}

			count := 0
			for _, result := range results {
				select {
				case <-ctx.Done():
					return
				case resultChan <- result:
					count++
				}
			}

			sourceCompleteChan <- models.SourceCompleteEvent{
				Source: srcName,
				Count:  count,
			}
		}(source, name)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(sourceCompleteChan)
	}()
}
