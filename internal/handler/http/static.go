package http

import (
	"net/http"
	"path/filepath"
)

// ServeUI serves the web UI
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path
	if r.URL.Path == "/" {
		http.ServeFile(w, r, filepath.Join("web", "templates", "index.html"))
		return
	}

	// For other paths, let the redirect handler take over
	h.RedirectURL(w, r)
}

// SetupStaticFiles configures static file serving
func SetupStaticFiles(mux *http.ServeMux) {
	// Serve static files (CSS, JS)
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
}
