package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

// StaticHandler returns an http.Handler that serves the embedded frontend files.
// It falls back to index.html for SPA client-side routing.
func StaticHandler() http.Handler {
	dist, _ := fs.Sub(frontendFS, "frontend/dist")
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		// Check if file exists
		if f, err := dist.Open(path[1:]); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// Fallback to index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
