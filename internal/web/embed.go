// Package web provides the embedded static frontend assets.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets
var assets embed.FS

// FileServer returns an http.Handler that serves the embedded assets. Routes
// to "/" and "/index.html" serve the index page; "/assets/" serves the static
// subdirectory.
func FileServer() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic("web: assets subdirectory not found: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
