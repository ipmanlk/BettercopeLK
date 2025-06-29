package sources

import (
	"context"
	"ipmanlk/bettercopelk/internal/models"
)

type Source interface {
	Name() string
	Search(ctx context.Context, req models.SearchRequest) ([]models.SearchResult, error)
	Download(ctx context.Context, url string) ([]byte, string, error)
	IsAvailable() bool
}

type Manager struct {
	sources map[string]Source
}

func NewManager() *Manager {
	return &Manager{
		sources: make(map[string]Source),
	}
}

func (m *Manager) RegisterSource(source Source) {
	m.sources[source.Name()] = source
}

func (m *Manager) GetSource(name string) (Source, bool) {
	source, exists := m.sources[name]
	return source, exists
}

func (m *Manager) GetAllSources() map[string]Source {
	return m.sources
}

func (m *Manager) GetAvailableSources() []string {
	var available []string
	for name, source := range m.sources {
		if source.IsAvailable() {
			available = append(available, name)
		}
	}
	return available
}
