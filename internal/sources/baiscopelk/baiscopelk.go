package baiscopelk

import (
	"context"
	"fmt"
	"io"
	"ipmanlk/bettercopelk/internal/htmlparser"
	"ipmanlk/bettercopelk/internal/models"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BaiscopeLK struct {
	client  *http.Client
	baseURL string
}

func New() *BaiscopeLK {
	return &BaiscopeLK{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://www.baiscope.lk",
	}
}

func (o *BaiscopeLK) Name() string {
	return "baiscopelk"
}

func (o *BaiscopeLK) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "HEAD", o.baseURL, nil)
	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (o *BaiscopeLK) Search(ctx context.Context, req models.SearchRequest) ([]models.SearchResult, error) {
	searchURL := fmt.Sprintf("%s/?s=%s",
		o.baseURL, url.QueryEscape(req.Query))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	results, err := o.parseSearchResults(resp.Body, req.Query)
	if err != nil {
		return nil, fmt.Errorf("error parsing search results: %w", err)
	}

	return results, nil
}

func (o *BaiscopeLK) Download(ctx context.Context, postURL string) ([]byte, string, error) {
	downloadURL, err := o.getDownloadURL(ctx, postURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get download URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download subtitle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read download content: %w", err)
	}
	filename := o.extractFilename(resp, downloadURL)

	return content, filename, nil
}

func (o *BaiscopeLK) getDownloadURL(ctx context.Context, postURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", postURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch post page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	doc, err := htmlparser.NewDocument(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	selector := "a[data-e-disable-page-transition=true]"
	downloadLink, exists := doc.Find(selector).First().Attr("href")
	if !exists {
		return "", fmt.Errorf("download link not found on page")
	}

	return downloadLink, nil
}

func (o *BaiscopeLK) parseSearchResults(body io.Reader, query string) ([]models.SearchResult, error) {
	doc, err := htmlparser.NewDocument(body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	var results []models.SearchResult

	doc.Find("article.elementor-post").Each(func(i int, s *htmlparser.Element) {
		link := s.Find("a.elementor-post__thumbnail__link, h5.elementor-post__title a").First()
		url, exists := link.Attr("href")
		if !exists {
			return
		}

		title := s.Find("h5.elementor-post__title").First().Text()
		title = strings.TrimSpace(title)

		if title == "" {
			return
		}

		if o.shouldIgnore(url, title) {
			return
		}

		result := models.SearchResult{
			Title:  title,
			URL:    url,
			Source: o.Name(),
		}

		results = append(results, result)
	})

	return results, nil
}

func (o *BaiscopeLK) shouldIgnore(url string, title string) bool {
	ignorePatterns := []string{"Collection"}
	for _, pattern := range ignorePatterns {
		if strings.Contains(title, pattern) || strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func (o *BaiscopeLK) extractFilename(resp *http.Response, downloadURL string) string {
	if filename := resp.Header.Get("X-Dlm-File-Name"); filename != "" {
		return filename
	}

	if cd := resp.Header.Get("Content-Disposition"); cd != "" && strings.Contains(cd, "filename=") {
		start := strings.Index(cd, "filename=")
		if start != -1 {
			start += 9
			filename := cd[start:]
			filename = strings.Trim(filename, `";'`)
			if filename != "" {
				return filename
			}
		}
	}

	return fmt.Sprintf("baiscope_subtitle_%d.zip", time.Now().Unix())
}
