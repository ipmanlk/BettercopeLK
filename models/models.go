package models

type SearchResult struct {
	Title   string `json:"title"`
	PostURL string `json:"postUrl"`
	Source  string `json:"source"`
}

type SearchSource struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

type SubtitleData struct {
	Filename string
	Content  []byte
}
