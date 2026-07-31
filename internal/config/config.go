// Package config provides application configuration loaded from environment
// variables with sensible defaults.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application-wide configuration.
type Config struct {
	// Port is the HTTP listen port.
	Port int
	// SourcesDir is the directory containing source YAML files.
	SourcesDir string
	// MaxBulkItems is the maximum number of items in a bulk download request.
	MaxBulkItems int
	// MaxBodyBytes is the maximum request body size for POST endpoints.
	MaxBodyBytes int64
	// MaxFileBytes is the maximum size of a single upstream file download.
	MaxFileBytes int64
	// MaxBulkTotalBytes is the maximum total uncompressed size of a bulk zip.
	MaxBulkTotalBytes int64

	// RateSearchRPM is the per-IP rate limit for search endpoints, in requests per minute.
	RateSearchRPM int
	// RateDownloadRPM is the per-IP rate limit for single download.
	RateDownloadRPM int
	// RateBulkRPM is the per-IP rate limit for bulk download.
	RateBulkRPM int
	// RateGlobalRPM is the global per-IP rate limit across all routes.
	RateGlobalRPM int

	// HTTPTimeout is the per-request timeout for upstream HTTP calls.
	HTTPTimeout time.Duration
	// HTTPIdleTimeout is the idle timeout for the upstream connection pool.
	HTTPIdleTimeout time.Duration

	// ReadHeaderTimeout is the http.Server read-header timeout.
	ReadHeaderTimeout time.Duration
	// ReadTimeout is the http.Server read timeout.
	ReadTimeout time.Duration
	// WriteTimeout is the http.Server write timeout.
	WriteTimeout time.Duration

	// SourceAvailabilityTTL is how long a source availability check result stays valid.
	SourceAvailabilityTTL time.Duration

	// TrustedProxies lists IPs or CIDRs whose X-Forwarded-For header is trusted
	// for rate-limit IP extraction.
	TrustedProxies []string
}

// Load reads configuration from environment variables. Missing variables fall
// back to sensible defaults.
func Load() *Config {
	return &Config{
		Port:                  envInt("PORT", 8080),
		SourcesDir:            envStr("SOURCES_DIR", "./sources"),
		MaxBulkItems:          envInt("MAX_BULK_ITEMS", 20),
		MaxBodyBytes:          envInt64("MAX_BODY_BYTES", 65536),           // 64 KB
		MaxFileBytes:          envInt64("MAX_FILE_BYTES", 52428800),        // 50 MB
		MaxBulkTotalBytes:     envInt64("MAX_BULK_TOTAL_BYTES", 524288000), // 500 MB
		RateSearchRPM:         envInt("RATE_SEARCH_RPM", 60),
		RateDownloadRPM:       envInt("RATE_DOWNLOAD_RPM", 30),
		RateBulkRPM:           envInt("RATE_BULK_RPM", 10),
		RateGlobalRPM:         envInt("RATE_GLOBAL_RPM", 120),
		HTTPTimeout:           envDuration("HTTP_TIMEOUT", 30*time.Second),
		HTTPIdleTimeout:       envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ReadHeaderTimeout:     envDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:           envDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:          envDuration("WRITE_TIMEOUT", 300*time.Second),
		SourceAvailabilityTTL: envDuration("SOURCE_AVAILABILITY_TTL", 60*time.Second),
		TrustedProxies:        envList("TRUSTED_PROXIES"),
	}
}

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envInt64(key string, def int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envList(key string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
