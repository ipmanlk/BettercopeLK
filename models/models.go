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
	URL    string
	Source Source
}

type SubtitleData struct {
	Filename string
	Content  []byte
}

type SubtitleRequest struct {
	PostURL string `json:"postUrl"`
	Source  Source `json:"source"`
}

type BulkSubtitleRequest struct {
	Data []SubtitleRequest `json:"data"`
}
