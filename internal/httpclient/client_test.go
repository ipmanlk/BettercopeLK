package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ipmanlk/bettercopelk/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := config.Load()
	c := New(cfg)
	if c == nil {
		t.Fatal("New returned nil")
	}
}

func TestDefaultUserAgent(t *testing.T) {
	cfg := config.Load()
	c := New(cfg)

	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if gotUA == "" {
		t.Fatal("User-Agent header was empty")
	}
	if len(gotUA) < 20 {
		t.Errorf("User-Agent suspiciously short: %q", gotUA)
	}
}

func TestClientGet(t *testing.T) {
	cfg := config.Load()
	c := New(cfg)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestClientRespectsContext(t *testing.T) {
	cfg := config.Load()
	cfg.HTTPTimeout = 30 * time.Second // longer than our test context
	c := New(cfg)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait until the client context should have expired.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected context deadline error, got nil")
	}
}

func TestClientTimeoutFromConfig(t *testing.T) {
	cfg := config.Load()
	cfg.HTTPTimeout = 10 * time.Millisecond
	c := New(cfg)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	_, err := c.Get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
