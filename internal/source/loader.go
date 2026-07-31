package source

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"ipmanlk/bettercopelk/internal/httpclient"
)

// LoadSources reads all .yaml and .yml files in dir, validates them, and
// returns a populated Registry. Files whose enabled field is false are skipped.
// The returned registry uses the given client for all HTTP calls and the ttl
// for availability caching.
func LoadSources(dir string, client *httpclient.Client, ttl time.Duration) (*Registry, error) {
	reg := NewRegistry(client, ttl)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading sources directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var cfg SourceConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		if !cfg.Enabled {
			continue
		}

		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("validating %s: %w", entry.Name(), err)
		}

		s := NewScraper(&cfg, client)
		if err := reg.Add(s); err != nil {
			return nil, fmt.Errorf("registering %s: %w", entry.Name(), err)
		}
	}

	return reg, nil
}
