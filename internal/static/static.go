package static

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed assets/*
var Assets embed.FS

func GetStaticFileServer() http.Handler {
	fsys, err := fs.Sub(Assets, "assets")
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path

		if urlPath == "/index.html" || urlPath == "/" {
			content, err := fs.ReadFile(fsys, "index.html")
			if err != nil {
				http.Error(w, "Could not read index.html", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(content)
			return
		}

		if strings.HasPrefix(urlPath, "/assets/") {
			assetPath := strings.TrimPrefix(urlPath, "/assets/")

			switch {
			case strings.HasSuffix(assetPath, ".css"):
				w.Header().Set("Content-Type", "text/css")
			case strings.HasSuffix(assetPath, ".js"):
				w.Header().Set("Content-Type", "application/javascript")
			}

			content, err := fs.ReadFile(fsys, assetPath)
			if err != nil {
				http.Error(w, "Asset not found: "+assetPath, http.StatusNotFound)
				return
			}

			w.Write(content)
			return
		}

		http.NotFound(w, r)
	})
}
