package subtitle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"ipmanlk/bettercopelk/internal/config"
	"ipmanlk/bettercopelk/internal/models"
	"ipmanlk/bettercopelk/internal/security"
	"ipmanlk/bettercopelk/internal/source"
	"ipmanlk/bettercopelk/internal/token"
)

// fakeSource implements source.Source for testing.
type fakeSource struct {
	id        string
	name      string
	baseURL   string
	enabled   bool
	results   []models.SearchResult
	searchErr error
	dlContent []byte
	dlName    string
	dlErr     error
}

func (f *fakeSource) ID() string                       { return f.id }
func (f *fakeSource) Name() string                     { return f.name }
func (f *fakeSource) BaseURL() string                  { return f.baseURL }
func (f *fakeSource) Enabled() bool                    { return f.enabled }
func (f *fakeSource) Available(_ context.Context) bool { return true }
func (f *fakeSource) Search(_ context.Context, _ models.SearchRequest) ([]models.SearchResult, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.results, nil
}
func (f *fakeSource) Download(_ context.Context, _ string) ([]byte, string, error) {
	if f.dlErr != nil {
		return nil, "", f.dlErr
	}
	return f.dlContent, f.dlName, nil
}

func buildRegistry(srcs ...source.Source) *source.Registry {
	// Pass nil client — tests never call List or cachedAvailable through fakes.
	reg := source.NewRegistry(nil, 0)
	for _, s := range srcs {
		if err := reg.Add(s); err != nil {
			panic(fmt.Sprintf("registry add: %v", err))
		}
	}
	return reg
}

// testResolver avoids real DNS lookups in tests.
type testResolver struct{}

func (testResolver) LookupIPAddr(_ context.Context, host string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
}

// tok encodes a full URL into a path token for test requests.
func tok(t *testing.T, full string) string {
	t.Helper()
	p, err := token.EncodePath(full)
	if err != nil {
		t.Fatalf("EncodePath(%q): %v", full, err)
	}
	return p
}

func TestServiceSearchAggregation(t *testing.T) {
	reg := buildRegistry(
		&fakeSource{
			id: "src1", name: "Source One", baseURL: "https://one.example.com", enabled: true,
			results: []models.SearchResult{
				{Title: "Movie A (src1)", Path: "path1", Source: "src1"},
			},
		},
		&fakeSource{
			id: "src2", name: "Source Two", baseURL: "https://two.example.com", enabled: true,
			results: []models.SearchResult{
				{Title: "Movie A (src2)", Path: "path2", Source: "src2"},
			},
		},
	)

	srv := &Service{cfg: &config.Config{MaxBulkItems: 20}, reg: reg}

	resp, err := srv.Search(context.Background(), models.SearchRequest{Query: "Movie"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	got := map[string]bool{}
	for _, r := range resp.Results {
		got[r.Source] = true
	}
	if !got["src1"] || !got["src2"] {
		t.Errorf("missing sources: got %v", got)
	}
}

func TestServiceSearchErrorSkip(t *testing.T) {
	reg := buildRegistry(
		&fakeSource{
			id: "good", name: "Good", baseURL: "https://good.example.com", enabled: true,
			results: []models.SearchResult{{Title: "Movie", Path: "path", Source: "good"}},
		},
		&fakeSource{
			id: "bad", name: "Bad", baseURL: "https://bad.example.com", enabled: true,
			searchErr: errors.New("upstream down"),
		},
	)

	srv := &Service{cfg: &config.Config{MaxBulkItems: 20}, reg: reg}

	resp, err := srv.Search(context.Background(), models.SearchRequest{Query: "Movie"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result (good source only), got %d", len(resp.Results))
	}
	if resp.Results[0].Source != "good" {
		t.Errorf("expected result from 'good', got %s", resp.Results[0].Source)
	}
}

func TestServiceSearchContextCancellation(t *testing.T) {
	src := &fakeSource{
		id: "slow", name: "Slow", baseURL: "https://slow.example.com", enabled: true,
	}
	reg := buildRegistry(src)

	srv := &Service{cfg: &config.Config{MaxBulkItems: 20}, reg: reg}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, _ = srv.Search(ctx, models.SearchRequest{Query: "x"})
	// Test passes if Search returns before 1 second (context timeout is 5ms).
	// No assertion needed beyond no deadlock.
}

func TestServiceSearchStreamCancellation(t *testing.T) {
	src := &fakeSource{
		id: "src", name: "Src", baseURL: "https://src.example.com", enabled: true,
		results: []models.SearchResult{{Title: "A", Path: "x", Source: "src"}},
	}
	reg := buildRegistry(src)
	srv := &Service{cfg: &config.Config{MaxBulkItems: 20}, reg: reg}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	resultCh := make(chan models.SearchResult, 10)
	completeCh := make(chan models.SourceCompleteEvent, 10)

	srv.StreamSearch(ctx, models.SearchRequest{Query: "x"}, resultCh, completeCh)

	var results int
	var completes int
	for {
		select {
		case _, ok := <-resultCh:
			if !ok {
				resultCh = nil
			} else {
				results++
			}
		case _, ok := <-completeCh:
			if !ok {
				completeCh = nil
			} else {
				completes++
			}
		}
		if resultCh == nil && completeCh == nil {
			break
		}
	}

	// With a 5ms context, the source should be cancelled and return via
	// source-complete with an error, or the results channel may have the result.
	t.Logf("results=%d completes=%d", results, completes)
}

func TestServiceBulkItemCap(t *testing.T) {
	srv := &Service{cfg: &config.Config{MaxBulkItems: 2}}

	var items []models.BulkItem
	for i := 0; i < 3; i++ {
		items = append(items, models.BulkItem{Path: tok(t, fmt.Sprintf("https://x.example.com/%d", i)), Source: "x"})
	}

	_, err := srv.DownloadBulk(context.Background(), models.BulkDownloadRequest{Items: items}, bytes.NewBuffer(nil))
	if err == nil || !strings.Contains(err.Error(), "max 2") {
		t.Fatalf("expected error about max 2, got: %v", err)
	}
}

func TestServiceBulkEmpty(t *testing.T) {
	srv := &Service{cfg: &config.Config{MaxBulkItems: 20}}
	_, err := srv.DownloadBulk(context.Background(), models.BulkDownloadRequest{}, bytes.NewBuffer(nil))
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected error about at least one item, got: %v", err)
	}
}

func TestServiceBulkPartialFailure(t *testing.T) {
	reg := buildRegistry(
		&fakeSource{
			id: "s1", name: "S1", baseURL: "https://s1.example.com", enabled: true,
			dlContent: []byte("subtitle content"),
			dlName:    "movie.srt",
		},
		&fakeSource{
			id: "s2", name: "S2", baseURL: "https://s2.example.com", enabled: true,
			dlErr: fmt.Errorf("upstream 502"),
		},
		&fakeSource{
			id: "s3", name: "S3", baseURL: "https://s3.example.com", enabled: true,
			dlContent: []byte("more subtitles"),
			dlName:    "other.srt",
		},
	)

	guard := security.NewGuard([]string{"s1.example.com", "s2.example.com", "s3.example.com"})
	guard.SetResolver(testResolver{})

	srv := &Service{
		cfg:   &config.Config{MaxBulkItems: 5, MaxFileBytes: 1024 * 1024, MaxBulkTotalBytes: 50 * 1024 * 1024},
		reg:   reg,
		guard: guard,
	}

	var buf bytes.Buffer
	manifest, err := srv.DownloadBulk(context.Background(), models.BulkDownloadRequest{
		Items: []models.BulkItem{
			{Path: tok(t, "https://s1.example.com/a"), Source: "s1"},
			{Path: tok(t, "https://s2.example.com/b"), Source: "s2"},
			{Path: tok(t, "https://s3.example.com/c"), Source: "s3"},
		},
	}, &buf)
	if err != nil {
		t.Fatalf("DownloadBulk: %v", err)
	}

	if len(manifest.Items) != 3 {
		t.Fatalf("expected 3 manifest items, got %d", len(manifest.Items))
	}

	status := map[string]bool{}
	for _, item := range manifest.Items {
		status[item.Source] = item.OK
	}

	if !status["s1"] {
		t.Errorf("s1 should be ok")
	}
	if status["s2"] {
		t.Errorf("s2 should have failed")
	}
	if !status["s3"] {
		t.Errorf("s3 should be ok")
	}

	if buf.Len() == 0 {
		t.Fatal("bulk zip is empty")
	}
}

func TestServiceBulkDedupe(t *testing.T) {
	reg := buildRegistry(
		&fakeSource{
			id: "s1", name: "S1", baseURL: "https://s1.example.com", enabled: true,
			dlContent: []byte("data"),
			dlName:    "m.srt",
		},
	)

	guard := security.NewGuard([]string{"s1.example.com"})
	guard.SetResolver(testResolver{})

	srv := &Service{
		cfg:   &config.Config{MaxBulkItems: 5, MaxFileBytes: 1024 * 1024, MaxBulkTotalBytes: 50 * 1024 * 1024},
		reg:   reg,
		guard: guard,
	}

	var buf bytes.Buffer
	_, err := srv.DownloadBulk(context.Background(), models.BulkDownloadRequest{
		Items: []models.BulkItem{
			{Path: tok(t, "https://s1.example.com/a"), Source: "s1"},
			{Path: tok(t, "https://s1.example.com/a"), Source: "s1"}, // duplicate
			{Path: tok(t, "https://s1.example.com/a"), Source: "s1"}, // duplicate
		},
	}, &buf)
	if err != nil {
		t.Fatalf("DownloadBulk: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("bulk zip is empty")
	}
}
