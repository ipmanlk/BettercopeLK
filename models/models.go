package models

type Source string

const (
	SourceBaiscopelk Source = "baiscopelk"
	SourceCineru     Source = "cineru"
	SourcePiratelk   Source = "piratelk"
)

type SearchResult struct {
	Title   string `json:"title"`
	PostURL string `json:"postUrl"`
	Source  Source `json:"source"`
}

type SearchSource struct {
	URL    string `json:"url"`
	Source Source `json:"name"`
}

type SubtitleData struct {
	Filename string
	Content  []byte
}
