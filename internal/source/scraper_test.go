package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/httpclient"
	"ipmanlk/bettercopelk/internal/models"
)

// testServer serves fixture HTML and optional download content on specific
// paths. It mirrors the routes a real subtitle site would have.
type testServer struct {
	server          *httptest.Server
	searchFixtures  map[string]string // path → HTML fixture
	postFixtures    map[string]string // path → HTML fixture
	downloadPath    string            // path where download is served
	downloadContent []byte
	downloadHeaders map[string]string
}

func newTestServer() *testServer {
	return &testServer{
		searchFixtures:  make(map[string]string),
		postFixtures:    make(map[string]string),
		downloadHeaders: make(map[string]string),
	}
}

func (ts *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := ts.searchFixtures[r.URL.Path]; ok && r.Method == http.MethodHead {
		// Availability check.
		w.WriteHeader(http.StatusOK)
		return
	}
	if html, ok := ts.searchFixtures[r.URL.Path]; ok {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}
	if html, ok := ts.postFixtures[r.URL.Path]; ok {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}
	if r.URL.Path == ts.downloadPath {
		for k, v := range ts.downloadHeaders {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Write(ts.downloadContent)
		return
	}
	http.NotFound(w, r)
}

func (ts *testServer) Start(t *testing.T) {
	ts.server = httptest.NewServer(ts)
	t.Cleanup(ts.server.Close)
}

func (ts *testServer) URL() string {
	return ts.server.URL
}

func TestScraper_Search(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		queryParam    string
		resultsSel    string
		titleSel      string
		linkSel       string
		linkAttr      string
		ignorePattern []string
		searchFixture string
		wantCount     int
		wantTitles    []string
	}{
		{
			name:       "baiscopelk-like search",
			id:         "baiscopelk",
			queryParam: "s",
			resultsSel: "article.elementor-post",
			titleSel:   "h5.elementor-post__title a",
			linkSel:    "a.elementor-post__thumbnail__link, h5.elementor-post__title a",
			linkAttr:   "href",
			searchFixture: `<html><body>
				<article class="elementor-post">
					<h5 class="elementor-post__title"><a href="/movie1">Movie 1 Subtitle</a></h5>
					<a class="elementor-post__thumbnail__link" href="/movie1"></a>
				</article>
				<article class="elementor-post">
					<h5 class="elementor-post__title"><a href="/movie2">Movie 2 Subtitle</a></h5>
					<a class="elementor-post__thumbnail__link" href="/movie2"></a>
				</article>
			</html>`,
			wantCount:  2,
			wantTitles: []string{"Movie 1 Subtitle", "Movie 2 Subtitle"},
		},
		{
			name:       "cineru-like search (row-based)",
			id:         "cineru",
			queryParam: "s",
			resultsSel: ".item-list .post-box-title a",
			titleSel:   "",
			linkSel:    "",
			linkAttr:   "href",
			searchFixture: `<html><body>
				<div class="item-list">
					<h3 class="post-box-title"><a href="/movie1">Movie 1 Subtitle</a></h3>
				</div>
				<div class="item-list">
					<h3 class="post-box-title"><a href="/movie2">Movie 2 Subtitle</a></h3>
				</div>
			</html>`,
			wantCount:  2,
			wantTitles: []string{"Movie 1 Subtitle", "Movie 2 Subtitle"},
		},
		{
			name:       "zoomlk-like search",
			id:         "zoomlk",
			queryParam: "s",
			resultsSel: ".td-ss-main-content .item-details .entry-title a",
			titleSel:   "",
			linkSel:    "",
			linkAttr:   "href",
			searchFixture: `<html><body>
				<div class="td-ss-main-content">
					<div class="item-details">
						<h3 class="entry-title"><a href="/movie1">Movie 1 Subtitle</a></h3>
					</div>
				</div>
			</html>`,
			wantCount:  1,
			wantTitles: []string{"Movie 1 Subtitle"},
		},
		{
			name:          "ignore patterns filter results",
			id:            "filtered",
			queryParam:    "s",
			resultsSel:    "article",
			titleSel:      "h2 a",
			linkSel:       "h2 a",
			linkAttr:      "href",
			ignorePattern: []string{"Collection"},
			searchFixture: `<html><body>
				<article><h2><a href="/movie1">Movie 1 Subtitle</a></h2></article>
				<article><h2><a href="/collection">Collection Title</a></h2></article>
				<article><h2><a href="/movie2">Movie 2 Subtitle</a></h2></article>
			</html>`,
			wantCount:  2,
			wantTitles: []string{"Movie 1 Subtitle", "Movie 2 Subtitle"},
		},
		{
			name:          "empty results when nothing matches",
			id:            "empty",
			queryParam:    "q",
			resultsSel:    ".result",
			titleSel:      "a",
			linkSel:       "a",
			linkAttr:      "href",
			searchFixture: `<html><body><div>No results</div></html>`,
			wantCount:     0,
			wantTitles:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer()
			srv.searchFixtures["/"] = tt.searchFixture
			srv.Start(t)

			// Create a scraper pointing at the test server.
			cfg := &SourceConfig{
				ID:      tt.id,
				Name:    tt.name,
				BaseURL: srv.URL(),
				Enabled: true,
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: tt.queryParam,
					Results: ResultsConfig{
						Selector: tt.resultsSel,
						Title:    FieldDef{Selector: tt.titleSel},
						Link:     FieldDef{Selector: tt.linkSel, Attribute: tt.linkAttr},
					},
					IgnorePatterns: tt.ignorePattern,
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
					},
					Method: "GET",
				},
			}

			client := newTestClient()
			scraper := NewScraper(cfg, client)

			results, err := scraper.Search(context.Background(), models.SearchRequest{Query: "test"})
			if err != nil {
				t.Fatalf("Search() returned error: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}

			for i, title := range tt.wantTitles {
				if i >= len(results) {
					break
				}
				if results[i].Title != title {
					t.Errorf("result[%d].Title = %q, want %q", i, results[i].Title, title)
				}
				if results[i].Source != tt.id {
					t.Errorf("result[%d].Source = %q, want %q", i, results[i].Source, tt.id)
				}
			}
		})
	}
}

func TestScraper_Download(t *testing.T) {
	downloadContent := []byte("mock subtitle content")

	tests := []struct {
		name                string
		method              string
		headers             map[string]string
		filenameHeader      string
		buttonSel           string
		buttonAttr          string
		postFixture         string
		downloadPath        string
		downloadRespHeaders map[string]string
		wantFilename        string
	}{
		{
			name:         "GET download with href",
			method:       "GET",
			buttonSel:    ".download-button",
			buttonAttr:   "href",
			postFixture:  `<html><body><a class="download-button" href="/subs/movie1.srt">DL</a></html>`,
			downloadPath: "/subs/movie1.srt",
			downloadRespHeaders: map[string]string{
				"Content-Disposition": `attachment; filename="Inception.srt"`,
			},
			wantFilename: "Inception.srt",
		},
		{
			name:                "GET download with data-link attribute",
			method:              "GET",
			buttonSel:           "#btn-download",
			buttonAttr:          "data-link",
			postFixture:         `<html><body><button id="btn-download" data-link="/dl/subtitle.zip">DL</button></html>`,
			downloadPath:        "/dl/subtitle.zip",
			downloadRespHeaders: map[string]string{},
			wantFilename:        "subtitle.zip",
		},
		{
			name:           "POST download with filename header",
			method:         "POST",
			filenameHeader: "X-Dlm-File-Name",
			headers:        map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			buttonSel:      "a.dl-link",
			buttonAttr:     "href",
			postFixture:    `<html><body><a class="dl-link" href="/dl/file.zip">DL</a></html>`,
			downloadPath:   "/dl/file.zip",
			downloadRespHeaders: map[string]string{
				"X-Dlm-File-Name": "Inception.zip",
			},
			wantFilename: "Inception.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer()
			srv.postFixtures["/post/1"] = tt.postFixture
			srv.downloadPath = tt.downloadPath
			srv.downloadContent = downloadContent
			srv.downloadHeaders = tt.downloadRespHeaders
			srv.Start(t)

			cfg := &SourceConfig{
				ID:      "test",
				Name:    tt.name,
				BaseURL: srv.URL(),
				Enabled: true,
				Search: SearchConfig{
					Path:       "/",
					Method:     "GET",
					QueryParam: "s",
					Results: ResultsConfig{
						Selector: ".result",
						Title:    FieldDef{Selector: "a"},
						Link:     FieldDef{Selector: "a", Attribute: "href"},
					},
				},
				Download: DownloadConfig{
					PostPage: PostPageConfig{
						Button: ButtonConfig{Selector: tt.buttonSel, Attribute: tt.buttonAttr},
					},
					Method:         tt.method,
					Headers:        tt.headers,
					FilenameHeader: tt.filenameHeader,
				},
			}

			client := newTestClient()
			scraper := NewScraper(cfg, client)

			postURL := srv.URL() + "/post/1"
			content, filename, err := scraper.Download(context.Background(), postURL)
			if err != nil {
				t.Fatalf("Download() returned error: %v", err)
			}

			if string(content) != string(downloadContent) {
				t.Errorf("Download() content = %q, want %q", string(content), string(downloadContent))
			}
			if filename != tt.wantFilename {
				t.Errorf("Download() filename = %q, want %q", filename, tt.wantFilename)
			}
		})
	}
}

func TestScraper_Download_Errors(t *testing.T) {
	t.Run("button not found on page", func(t *testing.T) {
		srv := newTestServer()
		srv.postFixtures["/post/1"] = `<html><body>No button here</body></html>`
		srv.Start(t)

		cfg := &SourceConfig{
			ID:      "test",
			Name:    "Test",
			BaseURL: srv.URL(),
			Enabled: true,
			Search: SearchConfig{
				Path:       "/",
				Method:     "GET",
				QueryParam: "q",
				Results: ResultsConfig{
					Selector: ".row",
					Title:    FieldDef{Selector: "a"},
					Link:     FieldDef{Selector: "a", Attribute: "href"},
				},
			},
			Download: DownloadConfig{
				PostPage: PostPageConfig{
					Button: ButtonConfig{Selector: ".download-btn", Attribute: "href"},
				},
				Method: "GET",
			},
		}

		client := newTestClient()
		scraper := NewScraper(cfg, client)

		_, _, err := scraper.Download(context.Background(), srv.URL()+"/post/1")
		if err == nil {
			t.Fatal("Download() expected error for missing button, got nil")
		}
	})

	t.Run("post page 404", func(t *testing.T) {
		srv := newTestServer()
		srv.Start(t)

		cfg := &SourceConfig{
			ID:      "test",
			Name:    "Test",
			BaseURL: srv.URL(),
			Enabled: true,
			Search: SearchConfig{
				Path:       "/",
				Method:     "GET",
				QueryParam: "q",
				Results: ResultsConfig{
					Selector: ".row",
					Title:    FieldDef{Selector: "a"},
					Link:     FieldDef{Selector: "a", Attribute: "href"},
				},
			},
			Download: DownloadConfig{
				PostPage: PostPageConfig{
					Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
				},
				Method: "GET",
			},
		}

		client := newTestClient()
		scraper := NewScraper(cfg, client)
		// Use a path the server doesn't handle.
		_, _, err := scraper.Download(context.Background(), srv.URL()+"/nonexistent")
		if err == nil {
			t.Fatal("Download() expected error for 404, got nil")
		}
	})
}

func TestScraper_Available(t *testing.T) {
	t.Run("available when HEAD returns 200", func(t *testing.T) {
		srv := newTestServer()
		srv.searchFixtures["/"] = "<html></html>" // serves both HEAD and GET
		srv.Start(t)

		cfg := &SourceConfig{
			ID:      "test",
			Name:    "Test",
			BaseURL: srv.URL(),
			Search: SearchConfig{
				Results: ResultsConfig{
					Selector: ".a",
					Title:    FieldDef{Selector: "a"},
					Link:     FieldDef{Selector: "a", Attribute: "href"},
				},
			},
			Download: DownloadConfig{
				PostPage: PostPageConfig{
					Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
				},
				Method: "GET",
			},
		}
		client := newTestClient()
		scraper := NewScraper(cfg, client)

		if !scraper.Available(context.Background()) {
			t.Error("Available() = false, want true")
		}
	})

	t.Run("unavailable when server refuses", func(t *testing.T) {
		// No server started — client will fail.
		cfg := &SourceConfig{
			ID:      "test",
			Name:    "Test",
			BaseURL: "http://127.0.0.1:1",
			Search: SearchConfig{
				Results: ResultsConfig{
					Selector: ".a",
					Title:    FieldDef{Selector: "a"},
					Link:     FieldDef{Selector: "a", Attribute: "href"},
				},
			},
			Download: DownloadConfig{
				PostPage: PostPageConfig{
					Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
				},
				Method: "GET",
			},
		}
		client := newTestClient()
		scraper := NewScraper(cfg, client)

		if scraper.Available(context.Background()) {
			t.Error("Available() = true, want false")
		}
	})
}

func TestRegistry_AddGet(t *testing.T) {
	client := newTestClient()
	reg := NewRegistry(client, 0)

	s := NewScraper(&SourceConfig{
		ID:      "test",
		Name:    "Test",
		BaseURL: "https://example.com",
		Search: SearchConfig{
			Path:       "/",
			Method:     "GET",
			QueryParam: "q",
			Results: ResultsConfig{
				Selector: ".row",
				Title:    FieldDef{Selector: "a"},
				Link:     FieldDef{Selector: "a", Attribute: "href"},
			},
		},
		Download: DownloadConfig{
			PostPage: PostPageConfig{
				Button: ButtonConfig{Selector: ".btn", Attribute: "href"},
			},
			Method: "GET",
		},
	}, client)

	if err := reg.Add(s); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	got := reg.Get("test")
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.ID() != "test" {
		t.Errorf("ID() = %q, want %q", got.ID(), "test")
	}

	// Duplicate add.
	if err := reg.Add(s); err == nil {
		t.Error("Add() expected error for duplicate, got nil")
	}

	// Get nonexistent.
	if reg.Get("nonexistent") != nil {
		t.Error("Get() for nonexistent should return nil")
	}
}

func TestRegistry_GetEnabled(t *testing.T) {
	client := newTestClient()
	reg := NewRegistry(client, 0)

	enabled := NewScraper(&SourceConfig{
		ID: "enabled", Name: "Enabled", BaseURL: "https://ex.com", Enabled: true,
		Search:   SearchConfig{Results: ResultsConfig{Selector: ".row", Title: FieldDef{Selector: "a"}, Link: FieldDef{Selector: "a", Attribute: "href"}}},
		Download: DownloadConfig{PostPage: PostPageConfig{Button: ButtonConfig{Selector: ".btn", Attribute: "href"}}, Method: "GET"},
	}, client)
	disabled := NewScraper(&SourceConfig{
		ID: "disabled", Name: "Disabled", BaseURL: "https://ex.com", Enabled: false,
		Search:   SearchConfig{Results: ResultsConfig{Selector: ".row", Title: FieldDef{Selector: "a"}, Link: FieldDef{Selector: "a", Attribute: "href"}}},
		Download: DownloadConfig{PostPage: PostPageConfig{Button: ButtonConfig{Selector: ".btn", Attribute: "href"}}, Method: "GET"},
	}, client)

	reg.Add(enabled)
	reg.Add(disabled)

	enabledList := reg.GetEnabled()
	if len(enabledList) != 1 {
		t.Errorf("GetEnabled() returned %d sources, want 1", len(enabledList))
	}
	if len(enabledList) > 0 && enabledList[0].ID() != "enabled" {
		t.Errorf("GetEnabled()[0].ID() = %q, want %q", enabledList[0].ID(), "enabled")
	}
}

func TestLoadSources(t *testing.T) {
	t.Run("load valid directory", func(t *testing.T) {
		client := newTestClient()
		reg, err := LoadSources("../../testdata/sources/valid", client, 0)
		if err != nil {
			t.Fatalf("LoadSources() error: %v", err)
		}
		if reg.Get("testsource") == nil {
			t.Error("LoadSources() did not load 'testsource'")
		}
	})

	t.Run("load nonexistent directory returns error", func(t *testing.T) {
		client := newTestClient()
		_, err := LoadSources("../../testdata/nonexistent", client, 0)
		if err == nil {
			t.Fatal("LoadSources() expected error, got nil")
		}
	})
}

// newTestClient creates an httpclient for testing with a short timeout.
func newTestClient() *httpclient.Client {
	cfg := config.Load()
	cfg.HTTPTimeout = 5 * time.Second
	return httpclient.New(cfg)
}
