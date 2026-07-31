// Package subtitle provides the domain service that orchestrates search and
// download operations across multiple subtitle sources.
package subtitle

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"ipmanlk/bettercopelk/internal/archive"
	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/httpclient"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/security"
	"ipmanlk/bettercopelk/internal/source"
)

const maxConcurrentFetches = 5

// Service orchestrates subtitle search and download across all configured
// sources.
type Service struct {
	cfg    *config.Config
	reg    *source.Registry
	guard  *security.Guard
	client *httpclient.Client
}

// New creates a Service with the given dependencies.
func New(cfg *config.Config, reg *source.Registry, guard *security.Guard, client *httpclient.Client) *Service {
	return &Service{cfg: cfg, reg: reg, guard: guard, client: client}
}

// Search fans out the query to all requested sources (or all enabled sources)
// and returns the aggregated results. A source error does not fail the entire
// search; errors are silently skipped.
func (s *Service) Search(ctx context.Context, req models.SearchRequest) (*models.SearchResponse, error) {
	sources := s.selectSources(req.Sources)

	var mu sync.Mutex
	var allResults []models.SearchResult

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrentFetches)

	for _, src := range sources {
		src := src
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}

			results, err := src.Search(ctx, req)
			if err != nil {
				return nil // skip source error
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	return &models.SearchResponse{Results: allResults}, nil
}

// StreamSearch fans out the query and streams results through channels. The
// caller owns closing both channels; StreamSearch closes them when all sources
// have responded or the context is cancelled.
func (s *Service) StreamSearch(ctx context.Context, req models.SearchRequest, resultCh chan<- models.SearchResult, completeCh chan<- models.SourceCompleteEvent) {
	sources := s.selectSources(req.Sources)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentFetches)

	for _, src := range sources {
		src := src
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				select {
				case completeCh <- models.SourceCompleteEvent{Source: src.ID(), Count: 0, Error: ctx.Err().Error()}:
				case <-ctx.Done():
				}
				return
			}

			results, err := src.Search(ctx, req)
			if err != nil {
				select {
				case completeCh <- models.SourceCompleteEvent{Source: src.ID(), Count: 0, Error: err.Error()}:
				case <-ctx.Done():
				}
				return
			}

			count := 0
			for _, r := range results {
				select {
				case resultCh <- r:
					count++
				case <-ctx.Done():
					return
				}
			}

			select {
			case completeCh <- models.SourceCompleteEvent{Source: src.ID(), Count: count}:
			case <-ctx.Done():
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultCh)
		close(completeCh)
	}()
}

// Download fetches a single subtitle file from the given source. It validates
// the URL through the security guard before making the request.
func (s *Service) Download(ctx context.Context, req models.DownloadRequest) ([]byte, string, error) {
	if err := s.guard.CheckURL(ctx, req.URL); err != nil {
		return nil, "", fmt.Errorf("%w", err)
	}

	src := s.reg.Get(req.Source)
	if src == nil {
		return nil, "", fmt.Errorf("source %q not found", req.Source)
	}

	return src.Download(ctx, req.URL)
}

// DownloadBulk fetches multiple subtitles in parallel, unzips any source zips,
// deduplicates filenames, and streams a single combined zip to w. Each item is
// validated through the security guard. Per-item errors are recorded in the
// manifest without aborting the entire download.
func (s *Service) DownloadBulk(ctx context.Context, req models.BulkDownloadRequest, w io.Writer) (*models.Manifest, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("bulk request must contain at least one item")
	}
	if len(req.Items) > s.cfg.MaxBulkItems {
		return nil, fmt.Errorf("bulk request contains %d items, max %d", len(req.Items), s.cfg.MaxBulkItems)
	}

	items := dedupeItems(req.Items)

	// Validate all URLs and sources upfront.
	for _, item := range items {
		if err := s.guard.CheckURL(ctx, item.URL); err != nil {
			return nil, fmt.Errorf("item %q: %w", item.URL, err)
		}
		if s.reg.Get(item.Source) == nil {
			return nil, fmt.Errorf("source %q not found", item.Source)
		}
	}

	sw := archive.NewWriter(w, s.cfg)
	defer sw.Close()

	type itemResult struct {
		item  models.BulkItem
		files []string
		bytes int64
		err   error
	}

	resultCh := make(chan itemResult, len(items))

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrentFetches)

	for _, item := range items {
		item := item
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}

			src := s.reg.Get(item.Source)
			content, filename, err := src.Download(ctx, item.URL)
			if err != nil {
				resultCh <- itemResult{item: item, err: err}
				return nil
			}

			if archive.IsZipFile(filename) {
				files, totalBytes, err := archive.ExtractAndCopy(s.cfg, sw, bytes.NewReader(content), int64(len(content)), "")
				if err != nil {
					resultCh <- itemResult{item: item, err: fmt.Errorf("extracting zip: %w", err)}
					return nil
				}
				resultCh <- itemResult{item: item, files: files, bytes: totalBytes}
			} else {
				name := filename
				if err := sw.AddFile(name, bytes.NewReader(content)); err != nil {
					resultCh <- itemResult{item: item, err: fmt.Errorf("adding file: %w", err)}
					return nil
				}
				resultCh <- itemResult{item: item, files: []string{name}, bytes: int64(len(content))}
			}
			return nil
		})
	}

	_ = g.Wait()
	close(resultCh)

	var manifestItems []models.ManifestItem
	for r := range resultCh {
		mi := models.ManifestItem{
			URL:    r.item.URL,
			Source: r.item.Source,
			OK:     r.err == nil,
			Files:  r.files,
			Bytes:  r.bytes,
		}
		if r.err != nil {
			mi.Error = r.err.Error()
		}
		manifestItems = append(manifestItems, mi)
	}

	manifest := &models.Manifest{
		Items:     manifestItems,
		CreatedAt: time.Now(),
	}

	if err := sw.AddManifest(manifest); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	if err := sw.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}

	return manifest, nil
}

// selectSources returns the list of sources to query. If ids is non-empty, it
// fetches each by id; otherwise it returns all enabled sources.
func (s *Service) selectSources(ids []string) []source.Source {
	if len(ids) == 0 {
		return s.reg.GetEnabled()
	}
	var out []source.Source
	for _, id := range ids {
		if src := s.reg.Get(id); src != nil {
			out = append(out, src)
		}
	}
	return out
}

// dedupeItems removes duplicate (url, source) pairs from the request.
func dedupeItems(items []models.BulkItem) []models.BulkItem {
	seen := make(map[string]bool)
	var out []models.BulkItem
	for _, item := range items {
		key := item.URL + "|" + item.Source
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
