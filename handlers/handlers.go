package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"ipmanlk/bettercopelk/download"
	"ipmanlk/bettercopelk/models"
	"ipmanlk/bettercopelk/search"
	"ipmanlk/bettercopelk/web"
)

func HandlePublicDirServe(w http.ResponseWriter, r *http.Request) {
	serverRoot, _ := fs.Sub(web.PublicFS, "public")
	fileServer := http.FileServer(http.FS(serverRoot))
	fileServer.ServeHTTP(w, r)
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set response headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get the query from the URL query string
	query := strings.TrimSpace(r.URL.Query().Get("query"))

	// if query is empty, send an Error event and close the connection
	if query == "" {
		_, err := w.Write([]byte("event: error\ndata: MISSING_QUERY\n\n"))
		if err != nil {
			log.Println("Error writing SSE error message:", err)
		}
		return
	}

	// Create a channel to send scraped results to the SSE client
	resultsChan := make(chan []models.SearchResult, 3)

	search.SearchSites(query, resultsChan)

	for results := range resultsChan {
		// Serialize the result as JSON
		resultJSON, err := json.Marshal(results)
		if err != nil {
			log.Println("Error marshaling JSON:", err)
			continue
		}

		// Wrap the JSON result in an SSE message
		sseMessage := "event: results\ndata: " + string(resultJSON) + "\n\n"

		_, err = w.Write([]byte(sseMessage))
		if err != nil {
			log.Println("Error writing SSE message:", err)
			return
		}

		w.(http.Flusher).Flush() // Flush the response to send it immediately
	}

	// Signal the end of the SSE stream
	endMessage := "event: end\ndata: end\n\n"
	_, err := w.Write([]byte(endMessage))
	if err != nil {
		log.Println("Error writing SSE end message:", err)
	}
}

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postUrl := strings.TrimSpace(r.URL.Query().Get("postUrl"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))

	if postUrl == "" {
		http.Error(w, "Missing link", http.StatusBadRequest)
		return
	}

	if source == "" {
		http.Error(w, "Missing source", http.StatusBadRequest)
		return
	}

	if !isValidSource(models.Source(source)) {
		http.Error(w, "Invalid source", http.StatusBadRequest)
		return
	}

	// send subtitle zip file
	data, err := download.GetSubtitle(postUrl, models.Source(source))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to download subtitle", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, data.Filename))

	// Write the file content to the response writer
	_, err = w.Write(data.Content)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func HandleBulkDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request data
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to parse the request", http.StatusBadRequest)
		return
	}

	var requestData models.BulkSubtitleRequest

	err = json.Unmarshal(body, &requestData)

	if err != nil {
		http.Error(w, "Failed to parse the request", http.StatusBadRequest)
		return
	}

	// Validate request data
	if len(requestData.Data) == 0 {
		http.Error(w, "Please provide at least one URL to download", http.StatusBadRequest)
		return
	}

	for _, subtitleRequest := range requestData.Data {
		if !isValidSource(models.Source(subtitleRequest.Source)) {
			http.Error(w, "Your request contains invalid sources", http.StatusBadRequest)
			return
		}
	}

	// Send merged zip file
	data, err := download.GetSubtitles(requestData.Data)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to download subtitles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, data.Filename))

	_, err = w.Write(data.Content)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// check if given source is valid
var validSources = map[models.Source]struct{}{
	models.SourceBaiscopelk: {},
	models.SourceCineru:     {},
	models.SourcePiratelk:   {},
}

func isValidSource(source models.Source) bool {
	_, exists := validSources[source]
	return exists
}
