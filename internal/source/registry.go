package source

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ipmanlk/bettercopelk/internal/httpclient"
)

type availabilityEntry struct {
	available bool
	checkedAt time.Time
}

// Registry holds all loaded subtitle sources and provides thread-safe lookup.
// Availability checks are cached with a configurable TTL so that a brief outage
// of a subtitle site does not immediately remove it from the sources list.
type Registry struct {
	mu      sync.RWMutex
	sources map[string]Source
	cache   map[string]availabilityEntry
	ttl     time.Duration
	client  *httpclient.Client
}

// NewRegistry creates an empty Registry. The ttl controls how long an
// availability check result stays valid.
func NewRegistry(client *httpclient.Client, ttl time.Duration) *Registry {
	return &Registry{
		sources: make(map[string]Source),
		cache:   make(map[string]availabilityEntry),
		ttl:     ttl,
		client:  client,
	}
}

// Add registers a Source. It returns an error if the id is already registered.
func (r *Registry) Add(s Source) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sources[s.ID()]; exists {
		return fmt.Errorf("source %q already registered", s.ID())
	}
	r.sources[s.ID()] = s
	return nil
}

// Get returns the Source for the given id, or nil if not found.
func (r *Registry) Get(id string) Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sources[id]
}

// GetAll returns a copy of all registered sources keyed by id.
func (r *Registry) GetAll() map[string]Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make(map[string]Source, len(r.sources))
	for k, v := range r.sources {
		cp[k] = v
	}
	return cp
}

// GetEnabled returns only sources whose Enabled() returns true. The YAML loader
// already filters disabled sources before Add, so this typically returns all
// registered sources.
func (r *Registry) GetEnabled() []Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Source
	for _, s := range r.sources {
		if s.Enabled() {
			out = append(out, s)
		}
	}
	return out
}

// List returns id + name for every registered source, with a cached
// availability flag for each.
func (r *Registry) List(ctx context.Context) []SourceInfo {
	srcs := r.GetAll()
	infos := make([]SourceInfo, 0, len(srcs))
	for _, s := range srcs {
		infos = append(infos, SourceInfo{
			ID:        s.ID(),
			Name:      s.Name(),
			Available: r.cachedAvailable(ctx, s),
		})
	}
	return infos
}

// cachedAvailable returns the cached availability result if it is still fresh,
// otherwise performs a live check.
func (r *Registry) cachedAvailable(ctx context.Context, s Source) bool {
	r.mu.RLock()
	entry, ok := r.cache[s.ID()]
	r.mu.RUnlock()

	if ok && time.Since(entry.checkedAt) < r.ttl {
		return entry.available
	}

	avail := s.Available(ctx)

	r.mu.Lock()
	r.cache[s.ID()] = availabilityEntry{available: avail, checkedAt: time.Now()}
	r.mu.Unlock()

	return avail
}

// SourceInfo is the public descriptor returned by /sources.
type SourceInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}
