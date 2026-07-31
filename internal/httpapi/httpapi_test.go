package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/httpclient"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/security"
	"ipmanlk/bettercopelk/internal/source"
	"ipmanlk/bettercopelk/internal/subtitle"
)

// ---------------------------------------------------------------------------
// testSource implements source.Source with configurable canned data.
// ---------------------------------------------------------------------------

type testSource struct {
	id         string
	name       string
	baseURL    string
	enabled    bool
	results    []models.SearchResult
	dlContent  []byte
	dlFilename string
	dlErr      error
}

func (s *testSource) ID() string                     { return s.id }
func (s *testSource) Name() string                   { return s.name }
func (s *testSource) BaseURL() string                { return s.baseURL }
func (s *testSource) Enabled() bool                  { return s.enabled }
func (s *testSource) Available(context.Context) bool { return true }
func (s *testSource) Search(context.Context, models.SearchRequest) ([]models.SearchResult, error) {
	return s.results, nil
}
func (s *testSource) Download(context.Context, string) ([]byte, string, error) {
	return s.dlContent, s.dlFilename, s.dlErr
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

type fakeResolver struct{}

func (fakeResolver) LookupIPAddr(ctx context.Context, host string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
}

func testServer(t *testing.T, src source.Source) *httptest.Server {
	t.Helper()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := httpclient.New(cfg)

	reg := source.NewRegistry(client, cfg.SourceAvailabilityTTL)
	if err := reg.Add(src); err != nil {
		t.Fatal(err)
	}

	enabled := reg.GetEnabled()
	providers := make([]security.BaseURLProvider, len(enabled))
	for i, s := range enabled {
		providers[i] = s
	}
	hosts := security.AllowedHosts(providers)
	guard := security.NewGuard(hosts)
	guard.SetResolver(fakeResolver{})

	svc := subtitle.New(cfg, reg, guard, client)
	srv := New(cfg, logger, reg, guard, svc, client)

	return httptest.NewServer(srv.Handler())
}

func testServerWithGuard(t *testing.T, g *security.Guard, srcs ...source.Source) *httptest.Server {
	t.Helper()
	g.SetResolver(fakeResolver{})

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := httpclient.New(cfg)

	reg := source.NewRegistry(client, cfg.SourceAvailabilityTTL)
	for _, src := range srcs {
		if err := reg.Add(src); err != nil {
			t.Fatal(err)
		}
	}

	svc := subtitle.New(cfg, reg, g, client)
	srv := New(cfg, logger, reg, g, svc, client)

	return httptest.NewServer(srv.Handler())
}

func mustGET(t *testing.T, baseURL, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustReadBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return b
}

// ---------------------------------------------------------------------------
// GET /healthz
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	src := &testSource{id: "test", name: "Test", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	b := mustReadBody(t, resp)
	var body map[string]string
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

// ---------------------------------------------------------------------------
// GET /readyz
// ---------------------------------------------------------------------------

func TestReadyz_Available(t *testing.T) {
	src := &testSource{id: "test", name: "Test", baseURL: "https://test.example", enabled: true, results: []models.SearchResult{}}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/readyz")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyz_Unavailable(t *testing.T) {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := httpclient.New(cfg)
	reg := source.NewRegistry(client, cfg.SourceAvailabilityTTL)

	guard := security.NewGuard(nil)
	svc := subtitle.New(cfg, reg, guard, client)
	srv := New(cfg, logger, reg, guard, svc, client)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mustGET(t, ts.URL, "/readyz")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/sources
// ---------------------------------------------------------------------------

func TestSources(t *testing.T) {
	src := &testSource{id: "testsrc", name: "Test Source", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/api/v1/sources")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	b := mustReadBody(t, resp)
	var out models.SourcesResponse
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(out.Sources))
	}
	if out.Sources[0].ID != "testsrc" {
		t.Errorf("expected testsrc, got %q", out.Sources[0].ID)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/search
// ---------------------------------------------------------------------------

func TestSearch_RequiresQuery(t *testing.T) {
	src := &testSource{id: "test", name: "Test", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/api/v1/search")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSearch_ReturnsResults(t *testing.T) {
	src := &testSource{
		id: "tsrc", name: "Test", baseURL: "https://test.example",
		enabled: true,
		results: []models.SearchResult{{Title: "Movie", URL: "https://test.example/post", Source: "tsrc"}},
	}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/api/v1/search?query=movie")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	b := mustReadBody(t, resp)
	var out models.SearchResponse
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].Title != "Movie" {
		t.Errorf("expected Movie, got %q", out.Results[0].Title)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/search/stream (SSE)
// ---------------------------------------------------------------------------

func TestSearchStream_EmitsEvents(t *testing.T) {
	src := &testSource{
		id: "ss", name: "SS", baseURL: "https://test.example",
		enabled: true,
		results: []models.SearchResult{
			{Title: "A", URL: "https://test.example/a", Source: "ss"},
			{Title: "B", URL: "https://test.example/b", Source: "ss"},
		},
	}
	srv := testServer(t, src)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/search/stream?query=movie")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected event-stream content-type, got %q", ct)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)

	// Should contain two result events and one source-complete event.
	if !strings.Contains(body, "event: result") {
		t.Error("missing result event")
	}
	if !strings.Contains(body, `"title":"A"`) {
		t.Error("missing first result data")
	}
	if !strings.Contains(body, `"title":"B"`) {
		t.Error("missing second result data")
	}
	if !strings.Contains(body, "event: source-complete") {
		t.Error("missing source-complete event")
	}
	if !strings.Contains(body, "event: done") {
		t.Error("missing done event")
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/download (single download)
// ---------------------------------------------------------------------------

func TestDownload_RequiresParams(t *testing.T) {
	src := &testSource{id: "t", name: "T", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	cases := []struct {
		path string
		code int
	}{
		{"/api/v1/download", 400},
		{"/api/v1/download?url=x", 400},
		{"/api/v1/download?source=t", 400},
	}
	for _, tc := range cases {
		resp := mustGET(t, srv.URL, tc.path)
		if resp.StatusCode != tc.code {
			t.Errorf("path %q: expected %d, got %d", tc.path, tc.code, resp.StatusCode)
		}
	}
}

func TestDownload_ReturnsFile(t *testing.T) {
	content := []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n")
	src := &testSource{
		id: "t", name: "T", baseURL: "https://test.example",
		enabled:    true,
		dlContent:  content,
		dlFilename: "test.srt",
	}
	srv := testServerWithGuard(t,
		security.NewGuard([]string{"test.example"}),
		src,
	)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/api/v1/download?url=https://test.example/file&source=t")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain, got %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="test.srt"`) {
		t.Errorf("unexpected Content-Disposition: %q", cd)
	}
	b := mustReadBody(t, resp)
	if !bytes.Equal(b, content) {
		t.Errorf("body mismatch: got %q, want %q", string(b), string(content))
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/download/bulk
// ---------------------------------------------------------------------------

func TestBulkDownload_ItemsLimit(t *testing.T) {
	cfg := config.Load()
	cfg.MaxBulkItems = 2

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := httpclient.New(cfg)
	reg := source.NewRegistry(client, cfg.SourceAvailabilityTTL)
	guard := security.NewGuard(nil)
	svc := subtitle.New(cfg, reg, guard, client)
	srv := New(cfg, logger, reg, guard, svc, client)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"items":[{"url":"1","source":"a"},{"url":"2","source":"a"},{"url":"3","source":"a"}]}`
	resp, err := http.Post(ts.URL+"/api/v1/download/bulk", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBulkDownload_StreamsZip(t *testing.T) {
	// Build a valid subtitle payload.
	srtContent := []byte("1\n00:00:01,000 --> 00:00:02,000\nSubtitle\n")

	src := &testSource{
		id: "bsrc", name: "BulkSrc", baseURL: "https://bsrc.example",
		enabled:    true,
		dlContent:  srtContent,
		dlFilename: "movie.srt",
	}

	srv := testServerWithGuard(t,
		security.NewGuard([]string{"bsrc.example"}),
		src,
	)
	defer srv.Close()

	body := `{"items":[{"url":"https://bsrc.example/subs","source":"bsrc"}]}`
	resp, err := http.Post(srv.URL+"/api/v1/download/bulk", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected application/zip, got %q", ct)
	}

	// Read the response as a zip file.
	raw := mustReadBody(t, resp)
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("reading zip response: %v", err)
	}

	foundSRT := false
	foundManifest := false
	for _, f := range zr.File {
		switch f.Name {
		case "movie.srt":
			foundSRT = true
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			if !bytes.Equal(b, srtContent) {
				t.Errorf("srt content mismatch")
			}
		case "manifest.json":
			foundManifest = true
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			var m models.Manifest
			if err := json.Unmarshal(b, &m); err != nil {
				t.Errorf("invalid manifest: %v", err)
			}
			if len(m.Items) != 1 || !m.Items[0].OK {
				t.Errorf("manifest item should be ok")
			}
		}
	}
	if !foundSRT {
		t.Error("missing movie.srt in zip")
	}
	if !foundManifest {
		t.Error("missing manifest.json in zip")
	}
}

// ---------------------------------------------------------------------------
// 404 for unknown routes
// ---------------------------------------------------------------------------

func TestUnknownRoute(t *testing.T) {
	src := &testSource{id: "t", name: "T", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	resp := mustGET(t, srv.URL, "/api/v1/nonexistent")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Request ID header propagation
// ---------------------------------------------------------------------------

func TestRequestIDHeader(t *testing.T) {
	src := &testSource{id: "t", name: "T", baseURL: "https://test.example", enabled: true}
	srv := testServer(t, src)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/healthz", nil)
	req.Header.Set("X-Request-ID", "my-id")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.Header.Get("X-Request-ID") != "my-id" {
		t.Errorf("expected X-Request-ID=my-id, got %q", resp.Header.Get("X-Request-ID"))
	}
}

// ---------------------------------------------------------------------------
// Rate limiting returns 429
// ---------------------------------------------------------------------------

func TestRateLimit(t *testing.T) {
	// Create a server with a very low rate limit so we can trigger 429.
	cfg := config.Load()
	cfg.RateSearchRPM = 1 // 1 request per minute
	cfg.RateGlobalRPM = 1

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := httpclient.New(cfg)

	src := &testSource{id: "rlsrc", name: "RL", baseURL: "https://test.example", enabled: true}
	reg := source.NewRegistry(client, cfg.SourceAvailabilityTTL)
	if err := reg.Add(src); err != nil {
		t.Fatal(err)
	}
	guard := security.NewGuard([]string{"test.example"})
	guard.SetResolver(fakeResolver{})
	svc := subtitle.New(cfg, reg, guard, client)

	srv := New(cfg, logger, reg, guard, svc, client)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Make 2 requests; the 2nd should get 429.
	resp1 := mustGET(t, ts.URL, "/api/v1/search?query=test")
	resp1.Body.Close()

	resp2 := mustGET(t, ts.URL, "/api/v1/search?query=test")
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()
}

// ---------------------------------------------------------------------------
// Content-type detection
// ---------------------------------------------------------------------------

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"zip", "file.zip", "application/zip"},
		{"srt", "file.srt", "text/plain; charset=utf-8"},
		{"ass", "file.ass", "text/plain; charset=utf-8"},
		{"ssa", "file.ssa", "text/plain; charset=utf-8"},
		{"vtt", "file.vtt", "text/vtt; charset=utf-8"},
		{"sub", "file.sub", "text/plain; charset=utf-8"},
		{"unknown", "file.xyz", "application/octet-stream"},
		{"no ext", "file", "application/octet-stream"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectContentType(tc.filename)
			if got != tc.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// splitSources
// ---------------------------------------------------------------------------

func TestSplitSources(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b", []string{"a", "b"}},
		{" a , b ", []string{"a", "b"}},
		{",,a,,", []string{"a"}},
	}
	for _, tc := range tests {
		got := splitSources(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitSources(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitSources(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}
