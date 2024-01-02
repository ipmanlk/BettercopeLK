package search

import (
	"context"
	"fmt"
	"ipmanlk/bettercopelk/models"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type sourceConfig struct {
	Source         models.Source
	URL            string
	Selector       string
	IgnorePatterns []string
}

var sourceConfigs = map[models.Source]sourceConfig{
	models.SourceBaiscopelk: {
		Source:         models.SourceBaiscopelk,
		URL:            "https://www.baiscope.lk?s=%s",
		Selector:       "article.post .entry-title a",
		IgnorePatterns: []string{"Collection"},
	},
	models.SourceCineru: {
		Source:         models.SourceCineru,
		URL:            "https://cineru.lk/?s=%s",
		Selector:       ".item-list .post-box-title a",
		IgnorePatterns: []string{"Collection"},
	},
	models.SourcePiratelk: {
		Source:         models.SourcePiratelk,
		URL:            "https://piratelk.com/?s=%s",
		Selector:       ".item-list .post-box-title a",
		IgnorePatterns: []string{"Collection"},
	},
	models.SourceZoomlk: {
		Source:         models.SourceZoomlk,
		URL:            "https://zoom.lk/?s=%s",
		Selector:       ".td-ss-main-content .item-details .entry-title a",
		IgnorePatterns: []string{"Collection"},
	},
}

func SearchSources(query string, resultsChan chan<- []models.SearchResult) {
	var wg sync.WaitGroup
	for _, config := range sourceConfigs {
		wg.Add(1)
		go func(cfg sourceConfig) {
			defer wg.Done()
			results, err := scrapeAndParseSource(cfg, query)
			if err != nil {
				log.Println(err)
				return
			}
			resultsChan <- results
		}(config)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()
}

func SearchSource(source models.Source, query string) ([]models.SearchResult, error) {
	cfg, exists := sourceConfigs[source]
	if !exists {
		return nil, fmt.Errorf("source %v not found", source)
	}

	return scrapeAndParseSource(cfg, query)
}

func scrapeAndParseSource(cfg sourceConfig, query string) ([]models.SearchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(cfg.URL, query), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []models.SearchResult = []models.SearchResult{}
	doc.Find(cfg.Selector).Each(func(_ int, s *goquery.Selection) {
		postURL, exists := s.Attr("href")
		if !exists {
			return
		}

		postTitle := strings.TrimSpace(s.Text())
		if postTitle == "" {
			return
		}

		if shouldIgnoreTitle(postTitle, cfg.IgnorePatterns) {
			return
		}

		results = append(results, models.SearchResult{Title: postTitle, PostURL: postURL, Source: cfg.Source})
	})
	return results, nil
}

func shouldIgnoreTitle(title string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(title, pattern) {
			return true
		}
	}
	return false
}
