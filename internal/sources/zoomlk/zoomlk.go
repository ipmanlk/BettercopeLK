package zoomlk

import (
	"context"
	"fmt"
	"io"
	"ipmanlk/bettercopelk/internal/htmlparser"
	"ipmanlk/bettercopelk/internal/models"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

type ZoomLK struct {
	client  *http.Client
	baseURL string
}

func New() *ZoomLK {
	return &ZoomLK{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://zoom.lk",
	}
}

func (z *ZoomLK) Name() string {
	return "zoomlk"
}

func (z *ZoomLK) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "HEAD", z.baseURL, nil)
	resp, err := z.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (z *ZoomLK) Search(ctx context.Context, req models.SearchRequest) ([]models.SearchResult, error) {
	searchURL := fmt.Sprintf("%s/?s=%s",
		z.baseURL, url.QueryEscape(req.Query))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := z.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	results, err := z.parseSearchResults(resp.Body, req.Query)
	if err != nil {
		return nil, fmt.Errorf("error parsing search results: %w", err)
	}

	return results, nil
}

func (z *ZoomLK) Download(ctx context.Context, postURL string) ([]byte, string, error) {
	downloadURL, err := z.getDownloadURL(ctx, postURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get download URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := z.client.Do(req)
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

	filename := z.extractFilename(resp, downloadURL)

	return content, filename, nil
}

func (z *ZoomLK) getDownloadURL(ctx context.Context, postURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", postURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := z.client.Do(req)
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

	selector := ".download-button"
	downloadLink, exists := doc.Find(selector).First().Attr("href")
	if !exists {
		return "", fmt.Errorf("download link not found on page")
	}

	return downloadLink, nil
}

func (z *ZoomLK) parseSearchResults(body io.Reader, query string) ([]models.SearchResult, error) {
	doc, err := htmlparser.NewDocument(body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	var results []models.SearchResult

	doc.Find(".td-ss-main-content .item-details .entry-title a").Each(func(i int, s *htmlparser.Element) {
		url, exists := s.Attr("href")
		if !exists {
			return
		}

		title := s.Text()
		title = strings.TrimSpace(title)

		if title == "" {
			return
		}

		if z.shouldIgnore(url, title) {
			return
		}

		result := models.SearchResult{
			Title:  title,
			URL:    url,
			Source: z.Name(),
		}

		results = append(results, result)
	})

	return results, nil
}

func (z *ZoomLK) shouldIgnore(url string, title string) bool {
	ignorePatterns := []string{"Collection"}
	for _, pattern := range ignorePatterns {
		if strings.Contains(title, pattern) || strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func (z *ZoomLK) extractFilename(resp *http.Response, downloadURL string) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if matches := regexp.MustCompile(`filename=["']?([^"']+)["']?`).FindStringSubmatch(cd); len(matches) > 1 {
			return matches[1]
		}
	}

	parsedURL, err := url.Parse(downloadURL)
	if err == nil {
		filename := path.Base(parsedURL.Path)
		if filename != "." && filename != "/" {
			return filename
		}
	}

	return fmt.Sprintf("zoomlk_subtitle_%d.zip", time.Now().Unix())
}
