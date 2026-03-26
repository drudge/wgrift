package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/drudge/wgrift/ui"
)

func (s *Server) staticHandler() http.Handler {
	// Serve from embedded assets (ui/web/ is embedded under "web/")
	webFS, err := fs.Sub(ui.WebAssets, "web")
	if err != nil {
		panic("embedded web assets missing: " + err.Error())
	}

	// Pre-read index.html for SPA fallback (avoids http.FileServer redirect issues)
	indexHTML, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		panic("embedded index.html missing: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to open the file from the embedded FS
		f, err := webFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			if strings.HasSuffix(path, ".wasm") {
				w.Header().Set("Content-Type", "application/wasm")
			}
			w.Header().Set("Cache-Control", "no-store")
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html directly for non-file routes
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(indexHTML)
	})
}
