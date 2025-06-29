package models

type SearchRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources,omitempty"`
}

type SearchResult struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type DownloadRequest struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

type SubtitleFile struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
}

type DownloadResponse struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	Size     int64  `json:"size"`
}

type SourcesResponse struct {
	Sources []string `json:"sources"`
}

type SourceCompleteEvent struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}
