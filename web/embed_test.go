package web

import (
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Text assets go out gzipped (and still decode to the original); clients
// that don't ask for gzip get the plain file.
func TestHandlerGzipsText(t *testing.T) {
	root, _ := fs.Sub(dist, "dist")
	entries, _ := fs.ReadDir(root, "assets")
	var js string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			js = "/assets/" + e.Name()
			break
		}
	}
	if js == "" {
		t.Skip("frontend not built")
	}
	raw, _ := fs.ReadFile(root, strings.TrimPrefix(js, "/"))
	h := Handler()

	req := httptest.NewRequest(http.MethodGet, js, nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("headers = %v", rec.Header())
	}
	if rec.Body.Len() >= len(raw) {
		t.Fatalf("gzipped %d bytes, raw %d", rec.Body.Len(), len(raw))
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(zr); string(got) != string(raw) {
		t.Fatal("gzipped body does not decode to the file")
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, js, nil))
	if rec.Header().Get("Content-Encoding") != "" || rec.Body.Len() != len(raw) {
		t.Fatalf("plain request: encoding %q, %d bytes", rec.Header().Get("Content-Encoding"), rec.Body.Len())
	}

	// SPA routes get index.html, compressed too.
	req = httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Header().Get("Content-Encoding") != "gzip" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("index: %d %v", rec.Code, rec.Header())
	}
}
