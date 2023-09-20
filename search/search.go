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

type parseFunction func(source string, doc *goquery.Document) []models.SearchResult

func SearchSites(query string, resultsChan chan<- []models.SearchResult) {
	sources := getSources(query)

	var wg sync.WaitGroup

	for _, source := range sources {
		wg.Add(1)

		praseFunc := parseGenericWpResponse
		if source.Name == "baiscopelk" {
			praseFunc = parseBaiscopeLkResponse
		}

		go scrapeSite(source.URL, source.Name, resultsChan, &wg, praseFunc)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()
}

func scrapeSite(url, source string, resultsChan chan<- []models.SearchResult, wg *sync.WaitGroup, parseFunc parseFunction) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	searchResults := parseFunc(source, doc)

	resultsChan <- searchResults
}

func parseBaiscopeLkResponse(source string, doc *goquery.Document) []models.SearchResult {
	searchResults := []models.SearchResult{}

	doc.Find("article.post").Each(func(i int, s *goquery.Selection) {
		entryLink := s.Find(".entry-title a")
		title := strings.TrimSpace(entryLink.Text())

		if title == "" || title == "Collection" {
			return
		}

		postURL, exists := entryLink.Attr("href")
		if !exists {
			return
		}

		searchResults = append(searchResults, models.SearchResult{Title: title, PostURL: postURL, Source: source})
	})

	return searchResults
}

func parseGenericWpResponse(source string, doc *goquery.Document) []models.SearchResult {
	searchResults := []models.SearchResult{}

	doc.Find(".item-list").Each(func(i int, s *goquery.Selection) {
		postBox := s.Find(".post-box-title a")
		title := strings.TrimSpace(postBox.Text())
		if title == "Collection" {
			return
		}

		postURL, exists := postBox.Attr("href")
		if !exists {
			return
		}

		searchResults = append(searchResults, models.SearchResult{Title: title, PostURL: postURL, Source: source})
	})

	return searchResults
}

func getSources(keyword string) []models.SearchSource {
	return []models.SearchSource{
		{
			URL:  fmt.Sprintf("https://www.baiscope.lk/?s=%s", keyword),
			Name: "baiscopelk",
		},
		{
			URL:  fmt.Sprintf("https://cineru.lk/?s=%s", keyword),
			Name: "cineru",
		},
		{
			URL:  fmt.Sprintf("https://piratelk.com/?s=%s", keyword),
			Name: "piratelk",
		},
	}
}
