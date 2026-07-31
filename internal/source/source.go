// Package source provides the pluggable source engine. Sources are defined in
// YAML files under sources/ — adding one requires no Go code or recompile.
package source

import (
	"context"

	"ipmanlk/bettercopelk/internal/models"
)

// A Source represents one subtitle site that can be searched and downloaded from.
type Source interface {
	// ID returns the stable identifier used in API parameters and YAML config.
	ID() string

	// Name returns the human-readable display name.
	Name() string

	// BaseURL returns the site root URL.
	BaseURL() string

	// Enabled reports whether this source should be included in searches.
	Enabled() bool

	// Available reports whether the source site is reachable. The result may be
	// cached by the caller.
	Available(ctx context.Context) bool

	// Search queries the source for subtitles matching req.Query.
	Search(ctx context.Context, req models.SearchRequest) ([]models.SearchResult, error)

	// Download fetches a subtitle file from the given post page URL. It returns
	// the raw file content and a suggested filename.
	Download(ctx context.Context, postURL string) ([]byte, string, error)
}
