package download

import (
	"ipmanlk/bettercopelk/models"
	"testing"
)

func TestDownloadLinkExtraction(t *testing.T) {
	tests := []struct {
		name   string
		source models.Source
		url    string
	}{
		{"Baiscopelk", models.SourceBaiscopelk, "https://www.baiscope.lk/phantom-2023-sinhala-subtitles/"},
		{"Cineru", models.SourceCineru, "https://cineru.lk/mangalavaaram-2023-sinhala-sub/"},
		{"Piratelk", models.SourcePiratelk, "https://piratelk.com/animal-2023-sinhala-subtitles/"},
		{"Zoomlk", models.SourceZoomlk, "https://zoom.lk/the-great-indian-family-2023/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getDownloadURL(sourceConfigs[tt.source], tt.url)
			if err != nil {
				t.Errorf("getDownloadURL() error = %v", err)
			}
		})
	}
}
