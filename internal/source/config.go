package source

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SourceConfig describes one subtitle source. It maps directly to a YAML file
// under sources/.
type SourceConfig struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	BaseURL  string         `yaml:"baseURL"`
	Enabled  bool           `yaml:"enabled"`
	Search   SearchConfig   `yaml:"search"`
	Download DownloadConfig `yaml:"download"`
}

// SearchConfig describes how to search a source.
type SearchConfig struct {
	Path           string            `yaml:"path"`
	Method         string            `yaml:"method"`
	QueryParam     string            `yaml:"queryParam"`
	Params         map[string]string `yaml:"params,omitempty"`
	Results        ResultsConfig     `yaml:"results"`
	IgnorePatterns []string          `yaml:"ignorePatterns,omitempty"`
}

// ResultsConfig describes how to extract search result rows from the search
// results HTML page.
type ResultsConfig struct {
	Selector string   `yaml:"selector"`
	Title    FieldDef `yaml:"title"`
	Link     FieldDef `yaml:"link"`
}

// FieldDef describes how to extract a single field from a search result row.
type FieldDef struct {
	Selector  string `yaml:"selector"`
	Attribute string `yaml:"attribute,omitempty"`
}

// DownloadConfig describes how to download a subtitle from a post page.
type DownloadConfig struct {
	PostPage       PostPageConfig    `yaml:"postPage"`
	Method         string            `yaml:"method"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	FilenameHeader string            `yaml:"filenameHeader,omitempty"`
}

// PostPageConfig describes how to find the download URL on the post page.
type PostPageConfig struct {
	Button ButtonConfig `yaml:"button"`
}

// ButtonConfig describes the download button element on a post page.
type ButtonConfig struct {
	Selector  string `yaml:"selector"`
	Attribute string `yaml:"attribute"`
}

var validID = regexp.MustCompile(`^[a-z0-9-]+$`)

// Validate checks the config for required fields and valid values. It returns
// a list of all problems or nil if the config is valid.
func (c *SourceConfig) Validate() error {
	var errs []string

	if c.ID == "" {
		errs = append(errs, "id is required")
	} else if !validID.MatchString(c.ID) {
		errs = append(errs, "id must match ^[a-z0-9-]+$")
	}

	if c.Name == "" {
		errs = append(errs, "name is required")
	}

	if c.BaseURL == "" {
		errs = append(errs, "baseURL is required")
	} else {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			errs = append(errs, fmt.Sprintf("baseURL: %v", err))
		} else if u.Scheme != "http" && u.Scheme != "https" {
			errs = append(errs, "baseURL scheme must be http or https")
		} else if u.Host == "" {
			errs = append(errs, "baseURL must have a host")
		}
	}

	if c.Search.Results.Selector == "" {
		errs = append(errs, "search.results.selector is required")
	}

	// If both title and link selectors are empty, we rely on the row element
	// itself. This is acceptable as long as the row selector targets the
	// anchor directly.

	if c.Download.PostPage.Button.Selector == "" {
		errs = append(errs, "download.postPage.button.selector is required")
	}

	if c.Download.Method != "" && c.Download.Method != "GET" && c.Download.Method != "POST" {
		errs = append(errs, "download.method must be GET or POST")
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
}
