// Package httpclient provides a shared HTTP client with retry, timeouts, and
// sensible connection-pool defaults.
package httpclient

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"ipmanlk/bettercopelk/internal/config"
)

// defaultUserAgent mimics a recent browser to avoid being blocked by subtitle
// sites that reject generic Go HTTP clients.
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// Client wraps retryablehttp.Client with a shared transport and convenience
// methods. All upstream HTTP calls in the application should go through this
// client.
type Client struct {
	*retryablehttp.Client
}

// New creates a Client from application config. It configures timeouts,
// connection pooling, user-agent, and retry policy from the config or defaults.
func New(cfg *config.Config) *Client {
	stdClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     cfg.HTTPIdleTimeout,
			DisableCompression:  false,
		},
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = stdClient
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 500 * time.Millisecond
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.Logger = nil // silence retryablehttp's own logging; we log at the middleware layer

	retryClient.CheckRetry = retryablehttp.DefaultRetryPolicy

	// Propagate the default user-agent on every request.
	retryClient.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, _ int) {
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", defaultUserAgent)
		}
	}

	return &Client{Client: retryClient}
}

// Do sends an HTTP request with retry and backoff. It converts a standard
// *http.Request into a retryable request internally.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	rreq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(rreq)
}

// Get is a convenience method for GET requests through the retryable client.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	rreq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(rreq)
}
