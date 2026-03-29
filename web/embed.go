// Package web embeds the compiled Vite/React frontend and provides an SPA handler
// that serves static files with a fallback to index.html for client-side routing.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var DistFS embed.FS

// spaHandler is an http.Handler that serves files from the embedded filesystem and
// falls back to index.html for any path that doesn't match an existing file.
// This enables React Router's client-side routing to work correctly.
type spaHandler struct {
	fs   http.FileSystem
	root fs.FS
}

// NewSPAHandler returns an http.Handler that serves the embedded SPA.
// Unknown paths (non-API/non-asset routes) fall back to index.html so the
// React Router can handle them client-side.
func NewSPAHandler() http.Handler {
	sub, err := fs.Sub(DistFS, "dist")
	if err != nil {
		panic("web: failed to sub dist/ from embed.FS: " + err.Error())
	}
	return &spaHandler{
		fs:   http.FS(sub),
		root: sub,
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Try to open the file — if it exists serve it directly.
	f, err := h.fs.Open(path)
	if err == nil {
		f.Close()
		http.FileServer(h.fs).ServeHTTP(w, r)
		return
	}

	// File not found: fall back to index.html for SPA client-side routing.
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/"
	http.FileServer(h.fs).ServeHTTP(w, r2)
}
