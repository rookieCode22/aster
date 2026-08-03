package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaHandler serves a client-side-rendered single-page application. Static
// assets that exist on disk (JS/CSS/images, hashed filenames from Vite) are
// served directly; any other GET path falls back to index.html so that
// client-side routes (e.g. /sessions, /chat/xyz) work on refresh and deep-link
// without a server-side 404. API and WebSocket paths are registered on separate
// more-specific patterns, so this handler only sees non-API routes.
type spaHandler struct {
	root  string // absolute path to the built frontend directory
	index string // index file name, typically "index.html"
}

func newSPAHandler(root, index string) *spaHandler {
	return &spaHandler{root: root, index: index}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	// Clean and join the requested path against the doc root, guarding against
	// directory traversal.
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	clean := filepath.Clean(filepath.FromSlash(upath))
	full := filepath.Join(h.root, clean)

	fi, err := os.Stat(full)
	if err == nil && !fi.IsDir() {
		// Real static file exists (e.g. /assets/index-HASH.js). Serve it.
		http.ServeFile(w, r, full)
		return
	}

	// Fall back to the SPA entry point for client-side routes.
	http.ServeFile(w, r, filepath.Join(h.root, h.index))
}
