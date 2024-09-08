package download

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"ipmanlk/bettercopelk/models"
)

type sourceConfig struct {
	Selector  string
	Attribute string
	Method    string // GET or POST
}

var sourceConfigs = map[models.Source]sourceConfig{
	models.SourceBaiscopelk: {
		Selector:  "a[data-e-disable-page-transition='true']",
		Attribute: "href",
		Method:    http.MethodPost,
	},
	models.SourceCineru: {
		Selector:  "#btn-download",
		Attribute: "data-link",
		Method:    http.MethodGet,
	},
	models.SourcePiratelk: {
		Selector:  ".download-button",
		Attribute: "href",
		Method:    http.MethodGet,
	},
	models.SourceZoomlk: {
		Selector:  ".download-button",
		Attribute: "href",
		Method:    http.MethodGet,
	},
}

func GetSubtitle(source models.Source, postUrl string) (*models.SubtitleData, error) {
	cfg := sourceConfigs[source]

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	link, err := getDownloadURL(cfg, postUrl)
	if err != nil {
		return nil, err
	}

	var filename string
	var downloadBody []byte
	var downloadErr error

	if cfg.Method == http.MethodPost {
		resp, err := client.Post(link, "application/x-www-form-urlencoded", nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		downloadBody, downloadErr = io.ReadAll(resp.Body)
		if downloadErr != nil {
			return nil, err
		}

		filename = getFilename(resp, link)
	} else {
		resp, err := client.Get(link)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		downloadBody, downloadErr = io.ReadAll(resp.Body)
		if downloadErr != nil {
			return nil, err
		}

		filename = getFilename(resp, link)
	}

	return &models.SubtitleData{
		Filename: filename,
		Content:  downloadBody,
	}, nil
}

func GetSubtitles(subtitleRequests []models.SubtitleRequest) (*models.SubtitleData, error) {
	var wg sync.WaitGroup

	resultCh := make(chan *models.SubtitleData, len(subtitleRequests))

	for _, request := range subtitleRequests {
		wg.Add(1)

		go func(request models.SubtitleRequest) {
			defer wg.Done()
			subData, err := GetSubtitle(request.Source, request.PostURL)

			if err != nil {
				return
			}

			resultCh <- subData
		}(request)
	}

	wg.Wait()
	close(resultCh)

	var subtitleData []*models.SubtitleData

	for subtitle := range resultCh {
		subtitleData = append(subtitleData, subtitle)
	}

	return createZipFile(subtitleData)
}

func getDownloadURL(cfg sourceConfig, postUrl string) (link string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, postUrl, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	link, exists := doc.Find(cfg.Selector).First().Attr(cfg.Attribute)
	if !exists {
		return "", fmt.Errorf("no download link found")
	}

	return link, nil
}

func createZipFile(subtitleData []*models.SubtitleData) (*models.SubtitleData, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, data := range subtitleData {
		// Create a new file header
		fileHeader := &zip.FileHeader{
			Name:   data.Filename,
			Method: zip.Deflate,
		}

		// Create a new file in the zip archive
		writer, err := zipWriter.CreateHeader(fileHeader)
		if err != nil {
			return nil, err
		}

		// Write the content of the file to the zip archive
		_, err = writer.Write(data.Content)
		if err != nil {
			return nil, err
		}
	}

	// Close the zip writer
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	// Create a new SubtitleData struct to hold all subtitles
	zippedData := models.SubtitleData{
		Filename: "bulk_subtitles.zip",
		Content:  buf.Bytes(),
	}

	return &zippedData, nil
}

func getFilename(resp *http.Response, downloadUrl string) string {
	validExtRegex := regexp.MustCompile(`\.(zip|rar|7z|tar)$`)

	// 1. check Content-Disposition header
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		standardRegex := regexp.MustCompile(`filename=["']([^"']+)["']`)
		extendedRegex := regexp.MustCompile(`filename\*\s*=\s*UTF-8''([^;]+)`)

		standardMatch := standardRegex.FindStringSubmatch(contentDisposition)
		if len(standardMatch) == 2 && validExtRegex.MatchString(standardMatch[1]) {
			return standardMatch[1]
		}

		extendedMatch := extendedRegex.FindStringSubmatch(contentDisposition)
		if len(extendedMatch) == 2 {
			decodedFilename, err := url.QueryUnescape(extendedMatch[1])
			if err == nil && validExtRegex.MatchString(decodedFilename) {
				return decodedFilename
			}
		}
	}

	// decode URL to handle any encoded characters
	decodedURL, err := url.QueryUnescape(downloadUrl)
	if err != nil {
		decodedURL = downloadUrl // Use the original URL if decoding fails
	}

	// 2. extract filename from URL's path
	parsedURL, err := url.Parse(decodedURL)
	if err == nil {
		pathFilename := path.Base(parsedURL.Path)
		if validExtRegex.MatchString(pathFilename) {
			return pathFilename
		}
	}

	// 3. Check URL Query Parameters
	for _, param := range parsedURL.Query() {
		for _, p := range param {
			if validExtRegex.MatchString(p) {
				return p
			}
		}
	}

	// 4. Default to "subtitle.zip"
	return "subtitle.zip"
}
