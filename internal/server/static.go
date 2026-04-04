package server

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/drudge/wgrift/ui"
)

// preloadedAsset holds a pre-read embedded file with its ETag.
type preloadedAsset struct {
	data []byte
	etag string
}

func preload(fsys fs.FS, name string) *preloadedAsset {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	hash := sha256.Sum256(data)
	return &preloadedAsset{
		data: data,
		etag: fmt.Sprintf(`"%x"`, hash[:8]),
	}
}

func (s *Server) staticHandler() http.Handler {
	webFS, err := fs.Sub(ui.WebAssets, "web")
	if err != nil {
		panic("embedded web assets missing: " + err.Error())
	}

	// Pre-read assets for fast serving and ETag computation.
	indexHTML := preload(webFS, "index.html")
	wasmRaw := preload(webFS, "wgrift.wasm")
	wasmGz := preload(webFS, "wgrift.wasm.gz")
	wasmBr := preload(webFS, "wgrift.wasm.br")

	fileServer := http.FileServer(http.FS(webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		trimmed := strings.TrimPrefix(path, "/")

		// Serve WASM with content negotiation and caching.
		if trimmed == "wgrift.wasm" {
			ae := r.Header.Get("Accept-Encoding")

			// Pick best compressed variant.
			var asset *preloadedAsset
			var encoding string
			if wasmBr != nil && strings.Contains(ae, "br") {
				asset = wasmBr
				encoding = "br"
			} else if wasmGz != nil && strings.Contains(ae, "gzip") {
				asset = wasmGz
				encoding = "gzip"
			} else {
				asset = wasmRaw
			}

			if asset == nil {
				http.NotFound(w, r)
				return
			}

			// ETag / conditional request.
			if match := r.Header.Get("If-None-Match"); match != "" && match == asset.etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Content-Type", "application/wasm")
			if encoding != "" {
				w.Header().Set("Content-Encoding", encoding)
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(asset.data)))
			w.Header().Set("ETag", asset.etag)
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Set("Cache-Control", "public, max-age=604800")
			w.Write(asset.data)
			return
		}

		// Serve index.html with revalidation caching.
		if trimmed == "index.html" {
			if indexHTML != nil {
				if match := r.Header.Get("If-None-Match"); match != "" && match == indexHTML.etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("ETag", indexHTML.etag)
				w.Header().Set("Cache-Control", "no-cache")
				w.Write(indexHTML.data)
				return
			}
		}

		// Other static files (wasm_exec.js, etc.) — serve with long cache.
		f, err := webFS.Open(trimmed)
		if err == nil {
			f.Close()
			w.Header().Set("Cache-Control", "public, max-age=604800")
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-file routes.
		if indexHTML != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(indexHTML.data)
			return
		}

		http.NotFound(w, r)
	})
}
