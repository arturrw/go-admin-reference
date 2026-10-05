// Package web embeds the built frontend (web/dist) into the Go binary and
// serves it as a single-page app.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

//go:embed all:dist
var dist embed.FS

// Text assets worth compressing; fonts and images already are.
var compressible = map[string]bool{".js": true, ".css": true, ".html": true, ".svg": true, ".json": true, ".txt": true}

// Handler serves static files from dist and falls back to index.html for
// client-side routes such as /products or /orders/10480. Text files are sent
// gzipped to clients that accept it; each file is compressed once and kept.
func Handler() http.Handler {
	root, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(root)
	_, statErr := fs.Stat(root, "index.html")
	built := statErr == nil
	gz := &gzipCache{root: root, files: map[string][]byte{}}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !built {
			http.Error(w, "frontend is not built: run `npm run build` in web/ (or use the Vite dev server on :5173)", http.StatusServiceUnavailable)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(root, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite emits content-hashed filenames, so they can be cached forever.
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				if gz.serve(w, r, name) {
					return
				}
				files.ServeHTTP(w, r)
				return
			}
			if path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		if gz.serve(w, r, "index.html") {
			return
		}
		http.ServeFileFS(w, r, root, "index.html")
	})
}

type gzipCache struct {
	root  fs.FS
	mu    sync.Mutex
	files map[string][]byte
}

// serve writes the gzipped file and reports true, or reports false when the
// client or the file type doesn't call for compression.
func (c *gzipCache) serve(w http.ResponseWriter, r *http.Request, name string) bool {
	ext := path.Ext(name)
	if !compressible[ext] || r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	w.Header().Add("Vary", "Accept-Encoding")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	body, err := c.get(name)
	if err != nil {
		return false
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(ext))
	w.Header().Set("Content-Encoding", "gzip")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body)) // embedded files have no mtime
	return true
}

func (c *gzipCache) get(name string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.files[name]; ok {
		return b, nil
	}
	raw, err := fs.ReadFile(c.root, name)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	c.files[name] = buf.Bytes()
	return c.files[name], nil
}
