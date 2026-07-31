package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	// Unset any env vars that might be set in the test environment.
	for _, key := range []string{
		"PORT", "SOURCES_DIR", "MAX_BULK_ITEMS", "MAX_BODY_BYTES",
		"MAX_FILE_BYTES", "MAX_BULK_TOTAL_BYTES",
		"RATE_SEARCH_RPM", "RATE_DOWNLOAD_RPM", "RATE_BULK_RPM", "RATE_GLOBAL_RPM",
		"HTTP_TIMEOUT", "HTTP_IDLE_TIMEOUT",
		"READ_HEADER_TIMEOUT", "READ_TIMEOUT", "WRITE_TIMEOUT",
		"SOURCE_AVAILABILITY_TTL", "TRUSTED_PROXIES",
	} {
		os.Unsetenv(key)
	}

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Port: got %d, want 8080", cfg.Port)
	}
	if cfg.SourcesDir != "./sources" {
		t.Errorf("SourcesDir: got %q, want %q", cfg.SourcesDir, "./sources")
	}
	if cfg.MaxBulkItems != 20 {
		t.Errorf("MaxBulkItems: got %d, want 20", cfg.MaxBulkItems)
	}
	if cfg.MaxBodyBytes != 65536 {
		t.Errorf("MaxBodyBytes: got %d, want 65536", cfg.MaxBodyBytes)
	}
	if cfg.MaxFileBytes != 52428800 {
		t.Errorf("MaxFileBytes: got %d, want 52428800", cfg.MaxFileBytes)
	}
	if cfg.MaxBulkTotalBytes != 524288000 {
		t.Errorf("MaxBulkTotalBytes: got %d, want 524288000", cfg.MaxBulkTotalBytes)
	}
	if cfg.RateSearchRPM != 60 {
		t.Errorf("RateSearchRPM: got %d, want 60", cfg.RateSearchRPM)
	}
	if cfg.RateDownloadRPM != 30 {
		t.Errorf("RateDownloadRPM: got %d, want 30", cfg.RateDownloadRPM)
	}
	if cfg.RateBulkRPM != 10 {
		t.Errorf("RateBulkRPM: got %d, want 10", cfg.RateBulkRPM)
	}
	if cfg.RateGlobalRPM != 120 {
		t.Errorf("RateGlobalRPM: got %d, want 120", cfg.RateGlobalRPM)
	}
	if cfg.HTTPTimeout != 30*time.Second {
		t.Errorf("HTTPTimeout: got %v, want 30s", cfg.HTTPTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("HTTPIdleTimeout: got %v, want 60s", cfg.HTTPIdleTimeout)
	}
	if cfg.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout: got %v, want 5s", cfg.ReadHeaderTimeout)
	}
	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout: got %v, want 10s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 300*time.Second {
		t.Errorf("WriteTimeout: got %v, want 300s", cfg.WriteTimeout)
	}
	if cfg.SourceAvailabilityTTL != 60*time.Second {
		t.Errorf("SourceAvailabilityTTL: got %v, want 60s", cfg.SourceAvailabilityTTL)
	}
	if cfg.TrustedProxies != nil {
		t.Errorf("TrustedProxies: got %v, want nil", cfg.TrustedProxies)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("SOURCES_DIR", "/etc/bettercopelk/sources")
	os.Setenv("MAX_BULK_ITEMS", "10")
	os.Setenv("MAX_BODY_BYTES", "4096")
	os.Setenv("MAX_FILE_BYTES", "1048576")
	os.Setenv("MAX_BULK_TOTAL_BYTES", "10485760")
	os.Setenv("RATE_SEARCH_RPM", "30")
	os.Setenv("RATE_DOWNLOAD_RPM", "15")
	os.Setenv("RATE_BULK_RPM", "5")
	os.Setenv("RATE_GLOBAL_RPM", "60")
	os.Setenv("HTTP_TIMEOUT", "60s")
	os.Setenv("HTTP_IDLE_TIMEOUT", "120s")
	os.Setenv("READ_HEADER_TIMEOUT", "10s")
	os.Setenv("READ_TIMEOUT", "15s")
	os.Setenv("WRITE_TIMEOUT", "600s")
	os.Setenv("SOURCE_AVAILABILITY_TTL", "30s")
	os.Setenv("TRUSTED_PROXIES", "10.0.0.1,10.0.0.2")
	defer func() {
		for _, key := range []string{
			"PORT", "SOURCES_DIR", "MAX_BULK_ITEMS", "MAX_BODY_BYTES",
			"MAX_FILE_BYTES", "MAX_BULK_TOTAL_BYTES",
			"RATE_SEARCH_RPM", "RATE_DOWNLOAD_RPM", "RATE_BULK_RPM", "RATE_GLOBAL_RPM",
			"HTTP_TIMEOUT", "HTTP_IDLE_TIMEOUT",
			"READ_HEADER_TIMEOUT", "READ_TIMEOUT", "WRITE_TIMEOUT",
			"SOURCE_AVAILABILITY_TTL", "TRUSTED_PROXIES",
		} {
			os.Unsetenv(key)
		}
	}()

	cfg := Load()

	if cfg.Port != 9090 {
		t.Errorf("Port: got %d, want 9090", cfg.Port)
	}
	if cfg.SourcesDir != "/etc/bettercopelk/sources" {
		t.Errorf("SourcesDir: got %q, want /etc/bettercopelk/sources", cfg.SourcesDir)
	}
	if cfg.MaxBulkItems != 10 {
		t.Errorf("MaxBulkItems: got %d, want 10", cfg.MaxBulkItems)
	}
	if cfg.MaxBodyBytes != 4096 {
		t.Errorf("MaxBodyBytes: got %d, want 4096", cfg.MaxBodyBytes)
	}
	if cfg.MaxFileBytes != 1048576 {
		t.Errorf("MaxFileBytes: got %d, want 1048576", cfg.MaxFileBytes)
	}
	if cfg.MaxBulkTotalBytes != 10485760 {
		t.Errorf("MaxBulkTotalBytes: got %d, want 10485760", cfg.MaxBulkTotalBytes)
	}
	if cfg.RateSearchRPM != 30 {
		t.Errorf("RateSearchRPM: got %d, want 30", cfg.RateSearchRPM)
	}
	if cfg.RateDownloadRPM != 15 {
		t.Errorf("RateDownloadRPM: got %d, want 15", cfg.RateDownloadRPM)
	}
	if cfg.RateBulkRPM != 5 {
		t.Errorf("RateBulkRPM: got %d, want 5", cfg.RateBulkRPM)
	}
	if cfg.RateGlobalRPM != 60 {
		t.Errorf("RateGlobalRPM: got %d, want 60", cfg.RateGlobalRPM)
	}
	if cfg.HTTPTimeout != 60*time.Second {
		t.Errorf("HTTPTimeout: got %v, want 60s", cfg.HTTPTimeout)
	}
	if cfg.HTTPIdleTimeout != 120*time.Second {
		t.Errorf("HTTPIdleTimeout: got %v, want 120s", cfg.HTTPIdleTimeout)
	}
	if cfg.SourceAvailabilityTTL != 30*time.Second {
		t.Errorf("SourceAvailabilityTTL: got %v, want 30s", cfg.SourceAvailabilityTTL)
	}
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "10.0.0.1" || cfg.TrustedProxies[1] != "10.0.0.2" {
		t.Errorf("TrustedProxies: got %v, want [10.0.0.1 10.0.0.2]", cfg.TrustedProxies)
	}
}

func TestInvalidEnvIntsFallBack(t *testing.T) {
	os.Setenv("PORT", "not-a-number")
	os.Setenv("MAX_BULK_ITEMS", "")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("MAX_BULK_ITEMS")
	}()

	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Port: got %d, want 8080", cfg.Port)
	}
	if cfg.MaxBulkItems != 20 {
		t.Errorf("MaxBulkItems: got %d, want 20", cfg.MaxBulkItems)
	}
}

func TestInvalidDurationFallsBack(t *testing.T) {
	os.Setenv("HTTP_TIMEOUT", "forever")
	defer os.Unsetenv("HTTP_TIMEOUT")

	cfg := Load()
	if cfg.HTTPTimeout != 30*time.Second {
		t.Errorf("HTTPTimeout: got %v, want 30s", cfg.HTTPTimeout)
	}
}
