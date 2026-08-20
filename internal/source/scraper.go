package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"ipmanlk/bettercopelk/internal/httpclient"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/token"
)

// contentDispositionRx extracts the filename from a Content-Disposition header.
// It matches filename="value", filename='value', and filename=value.
var contentDispositionRx = regexp.MustCompile(`filename=["']?([^"']+)["']?`)

// Scraper implements the Source interface using a SourceConfig to drive all
// site-specific behavior. There is one Scraper per YAML config file.
type Scraper struct {
	cfg    *SourceConfig
	client *httpclient.Client
}

// NewScraper creates a Scraper from its config and an HTTP client.
func NewScraper(cfg *SourceConfig, client *httpclient.Client) *Scraper {
	return &Scraper{cfg: cfg, client: client}
}

func (s *Scraper) ID() string {
	return s.cfg.ID
}

func (s *Scraper) Name() string {
	return s.cfg.Name
}

func (s *Scraper) BaseURL() string {
	return s.cfg.BaseURL
}

func (s *Scraper) Enabled() bool {
	return s.cfg.Enabled
}

// Available checks whether the source site is reachable by sending a HEAD
// request to the base URL.
func (s *Scraper) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.cfg.BaseURL, nil)
	if err != nil {
		return false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Search queries the source and returns matching subtitles.
func (s *Scraper) Search(ctx context.Context, req models.SearchRequest) ([]models.SearchResult, error) {
	searchURL := s.buildSearchURL(req.Query)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	return s.parseSearchResults(resp.Body)
}

// buildSearchURL constructs the search URL from the config.
func (s *Scraper) buildSearchURL(query string) string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	p := s.cfg.Search.Path
	if p == "" {
		p = "/"
	}
	u := base + p

	v := url.Values{}
	v.Set(s.cfg.Search.QueryParam, query)
	for k, val := range s.cfg.Search.Params {
		v.Set(k, val)
	}

	return u + "?" + v.Encode()
}

// parseSearchResults extracts search result rows from the HTML body using the
// configured selectors.
func (s *Scraper) parseSearchResults(body io.Reader) ([]models.SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	var results []models.SearchResult

	doc.Find(s.cfg.Search.Results.Selector).Each(func(_ int, row *goquery.Selection) {
		title := s.extractField(row, s.cfg.Search.Results.Title)
		link := s.extractField(row, s.cfg.Search.Results.Link)

		if title == "" || link == "" {
			return
		}

		if s.shouldIgnore(link, title) {
			return
		}

		pathTok, err := token.EncodePath(s.resolveURL(link))
		if err != nil {
			// Un-encodable link (e.g. non-http scheme) — skip the row.
			return
		}

		results = append(results, models.SearchResult{
			Title:  title,
			Path:   pathTok,
			Source: s.cfg.ID,
		})
	})

	return results, nil
}

// extractField extracts a text or attribute value from a row. If the field's
// selector is empty, the row itself is used.
func (s *Scraper) extractField(row *goquery.Selection, f FieldDef) string {
	var el *goquery.Selection
	if f.Selector == "" {
		el = row
	} else {
		el = row.Find(f.Selector).First()
	}

	if el.Length() == 0 {
		return ""
	}

	if f.Attribute != "" {
		return strings.TrimSpace(el.AttrOr(f.Attribute, ""))
	}
	return strings.TrimSpace(el.Text())
}

// shouldIgnore returns true if the title or URL contains any of the configured
// ignore patterns.
func (s *Scraper) shouldIgnore(url, title string) bool {
	for _, p := range s.cfg.Search.IgnorePatterns {
		if strings.Contains(url, p) || strings.Contains(title, p) {
			return true
		}
	}
	return false
}

// Download fetches a subtitle file given the URL of its post page.
func (s *Scraper) Download(ctx context.Context, postURL string) ([]byte, string, error) {
	downloadURL, err := s.findDownloadURL(ctx, postURL)
	if err != nil {
		return nil, "", fmt.Errorf("finding download URL: %w", err)
	}

	content, filename, err := s.fetchFile(ctx, downloadURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetching file: %w", err)
	}

	return content, filename, nil
}

// resolveURL converts a possibly-relative URL to absolute using the source's
// base URL.
func (s *Scraper) resolveURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	if strings.HasPrefix(rawURL, "/") {
		return base + rawURL
	}
	return base + "/" + rawURL
}

// findDownloadURL fetches the post page and extracts the download button URL.
func (s *Scraper) findDownloadURL(ctx context.Context, postURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, postURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating post page request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching post page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("post page returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parsing post page HTML: %w", err)
	}

	btn := s.cfg.Download.PostPage.Button
	sel := doc.Find(btn.Selector).First()
	if sel.Length() == 0 {
		return "", fmt.Errorf("download button %q not found on page", btn.Selector)
	}

	attr := btn.Attribute
	if attr == "" {
		attr = "href"
	}
	downloadURL, exists := sel.Attr(attr)
	if !exists || downloadURL == "" {
		return "", fmt.Errorf("download button attribute %q not found", attr)
	}

	return s.resolveURL(downloadURL), nil
}

// fetchFile downloads the actual subtitle file from the resolved download URL.
func (s *Scraper) fetchFile(ctx context.Context, downloadURL string) ([]byte, string, error) {
	var resp *http.Response
	var err error

	method := s.cfg.Download.Method
	if method == "" {
		method = http.MethodGet
	}

	var req *http.Request
	switch method {
	case http.MethodPost:
		req, err = http.NewRequestWithContext(ctx, method, downloadURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("creating POST request: %w", err)
		}
		for k, v := range s.cfg.Download.Headers {
			req.Header.Set(k, v)
		}
	default:
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("creating GET request: %w", err)
		}
		for k, v := range s.cfg.Download.Headers {
			req.Header.Set(k, v)
		}
	}

	resp, err = s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading download response: %w", err)
	}

	filename := s.extractFilename(resp, downloadURL)

	return content, filename, nil
}

// extractFilename determines the filename from the response or falls back to
// a generated name. The priority is:
//
//  1. The configured custom filename header (e.g. X-Dlm-File-Name).
//  2. The Content-Disposition header.
//  3. The basename of the download URL.
//  4. A fallback template.
func (s *Scraper) extractFilename(resp *http.Response, downloadURL string) string {
	// 1. Custom filename header.
	if h := s.cfg.Download.FilenameHeader; h != "" {
		if v := resp.Header.Get(h); v != "" {
			return v
		}
	}

	// 2. Content-Disposition header.
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if m := contentDispositionRx.FindStringSubmatch(cd); len(m) > 1 && m[1] != "" {
			return m[1]
		}
	}

	// 3. URL path basename.
	if u, err := url.Parse(downloadURL); err == nil {
		if base := path.Base(u.Path); base != "." && base != "/" && base != "" {
			return base
		}
	}

	// 4. Fallback.
	return fmt.Sprintf("%s_subtitle_%d.zip", s.cfg.ID, time.Now().UnixMilli())
}
