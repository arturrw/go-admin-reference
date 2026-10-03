// Package media stores uploaded product images on disk and renders the
// generated SVG artwork used for seed products.
package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MaxUploadBytes = 5 << 20

// Only raster formats are accepted: SVG uploads could carry scripts.
var allowed = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

var ErrUnsupportedType = errors.New("unsupported image type: use JPEG, PNG, WebP or GIF")

type Storage struct {
	dir string
}

func NewStorage(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Storage{dir: dir}, nil
}

// Save sniffs the content type, writes the file under a random name and
// returns that name and the number of bytes written.
func (s *Storage) Save(r io.Reader) (name string, size int64, err error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", 0, fmt.Errorf("read upload: %w", err)
	}
	head = head[:n]
	ext, ok := allowed[http.DetectContentType(head)]
	if !ok {
		return "", 0, ErrUnsupportedType
	}

	var rnd [12]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", 0, err
	}
	name = hex.EncodeToString(rnd[:]) + ext
	f, err := os.OpenFile(filepath.Join(s.dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	size, err = io.Copy(f, io.MultiReader(strings.NewReader(string(head)), io.LimitReader(r, MaxUploadBytes)))
	if err == nil && size > MaxUploadBytes {
		err = fmt.Errorf("file exceeds %d MB", MaxUploadBytes>>20)
	}
	if err != nil {
		f.Close()
		os.Remove(filepath.Join(s.dir, name))
		return "", 0, err
	}
	return name, size, nil
}

// Delete removes an uploaded file. Names are validated so a crafted value
// cannot escape the upload directory.
func (s *Storage) Delete(name string) error {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return fmt.Errorf("invalid file name %q", name)
	}
	err := os.Remove(filepath.Join(s.dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Handler serves uploaded files.
func (s *Storage) Handler() http.Handler {
	fs := http.FileServer(http.Dir(s.dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r) // no directory listings
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		fs.ServeHTTP(w, r)
	})
}
