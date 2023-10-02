package download

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"ipmanlk/bettercopelk/models"
)

type parseFunction func(doc *goquery.Document) (fileUrl string, err error)

func GetSubtitle(postUrl string, source models.Source) (*models.SubtitleData, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	link, err := getDownloadLink(postUrl, source)
	if err != nil {
		return nil, err
	}

	var filename string
	var downloadBody []byte
	var downloadErr error

	if source == models.SourceBaiscopelk {
		resp, err := client.Post(link, "application/x-www-form-urlencoded", nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		downloadBody, downloadErr = io.ReadAll(resp.Body)
		if downloadErr != nil {
			return nil, err
		}

		contentDisposition := resp.Header.Get("Content-Disposition")
		filename = getFilenameFromHeader(contentDisposition)

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

		filename = getFilenameFromURL(link)
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
			subData, err := GetSubtitle(request.PostURL, request.Source)

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

func getDownloadLink(postUrl string, source models.Source) (link string, err error) {
	parseFunc := parseBaiscopeLink

	switch source {
	case models.SourceCineru:
		parseFunc = parseCineruLink
	case models.SourcePiratelk:
		parseFunc = parsePiratelkLink
	}

	return parseDownloadLink(postUrl, parseFunc)
}

func parseBaiscopeLink(doc *goquery.Document) (link string, err error) {
	dLink, exists := doc.Find("img[src='https://baiscopelk.com/download.png']").Parent().Attr("href")
	if !exists {
		return "", fmt.Errorf("Download link not found")
	}
	return dLink, nil
}

func parseCineruLink(doc *goquery.Document) (link string, err error) {
	dLink, exists := doc.Find("#btn-download").Attr("data-link")
	if !exists {
		return "", fmt.Errorf("Download link not found")
	}
	return dLink, nil
}

func parsePiratelkLink(doc *goquery.Document) (link string, err error) {
	dLink, exists := doc.Find(".download-button").Attr("href")
	if !exists {
		return "", fmt.Errorf("Download link not found")
	}
	return dLink, nil
}

func parseDownloadLink(postUrl string, parseFunc parseFunction) (link string, err error) {
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

	return parseFunc(doc)
}

func getFilenameFromURL(fileUrl string) string {
	parts := strings.Split(strings.TrimSuffix(fileUrl, "/"), "/")
	filename := parts[len(parts)-1]

	if filename == "" {
		return "subtitle.zip"
	}

	if strings.Contains(filename, "?") {
		filename = parts[len(parts)-2]
	}

	if !strings.Contains(filename, ".") {
		filename += ".zip"
	}

	return filename
}

func getFilenameFromHeader(header string) string {
	headerRegex := regexp.MustCompile(`filename=["']([^"']+)["']`)
	match := headerRegex.FindStringSubmatch(header)
	if len(match) == 2 {
		return match[1]
	}
	return "subtitle.zip"
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
