package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	"github.com/arturrw/go-admin-reference/internal/httpapi"
	"github.com/arturrw/go-admin-reference/internal/media"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
	"github.com/arturrw/go-admin-reference/internal/seed"
	"github.com/arturrw/go-admin-reference/internal/store/memory"
	"github.com/arturrw/go-admin-reference/internal/store/postgres"
)

// A 1×1 transparent PNG.
var tinyPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

// newStore returns the in-memory store, or a freshly migrated and seeded
// Postgres database when TEST_DATABASE_URL is set (see `make test-pg`).
func newStore(t *testing.T, now time.Time) (httpapi.Store, httpapi.SessionStore) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		return memory.New(now), auth.NewMemorySessions(time.Hour)
	}
	ctx := context.Background()
	pool, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.SeedIfEmpty(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	return postgres.New(pool), postgres.NewSessions(pool, time.Hour)
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	uploads, err := media.NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	store, sessions := newStore(t, now)
	h := httpapi.New(httpapi.Deps{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Store:        store,
		Requests:     reqlog.New(500),
		Sessions:     sessions,
		Media:        uploads,
		Env:          "development",
		StartedAt:    now,
		DemoPassword: seed.DemoPassword,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

type client struct {
	t   *testing.T
	srv *httptest.Server
	c   *http.Client
}

func newClient(t *testing.T, srv *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, srv: srv, c: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (c *client) login(email string) {
	c.t.Helper()
	if code, body := c.do("POST", "/api/v1/auth/login", map[string]string{"email": email, "password": seed.DemoPassword}); code != 200 {
		c.t.Fatalf("login %s: %d %v", email, code, body)
	}
}

func TestAuthRequired(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	if code, _ := c.do("GET", "/api/v1/products", nil); code != http.StatusUnauthorized {
		t.Fatalf("want 401 without session, got %d", code)
	}
	if code, _ := c.do("POST", "/api/v1/auth/login", map[string]string{"email": "artur@acme.io", "password": "nope"}); code != http.StatusUnauthorized {
		t.Fatalf("want 401 for wrong password, got %d", code)
	}
	if code, _ := c.do("POST", "/api/v1/auth/login", map[string]string{"email": "lena@acme.io", "password": seed.DemoPassword}); code != http.StatusForbidden {
		t.Fatalf("suspended member must not sign in, got %d", code)
	}
	c.login("artur@acme.io")
	code, me := c.do("GET", "/api/v1/auth/me", nil)
	if code != 200 || me["user"].(map[string]any)["name"] != "Artur DCS" {
		t.Fatalf("me: %d %v", code, me)
	}
	c.do("POST", "/api/v1/auth/logout", nil)
	if code, _ := c.do("GET", "/api/v1/auth/me", nil); code != http.StatusUnauthorized {
		t.Fatalf("want 401 after logout, got %d", code)
	}
}

// TestRoleMatrix checks representative endpoints for every role.
func TestRoleMatrix(t *testing.T) {
	srv := newServer(t)
	// An invalid product: writers get past authorization and hit validation (422),
	// everyone else is stopped at 403. Keeps the test free of side effects.
	product := map[string]any{"name": "", "sku": "TST-1", "category": "Audio", "priceCents": 1000, "stock": 1, "status": "draft"}
	cases := []struct {
		method, path string
		body         any
		want         map[string]int // email → status
	}{
		{"GET", "/api/v1/products", nil, map[string]int{"artur@acme.io": 200, "mark@acme.io": 200, "yuki@acme.io": 200, "priya@acme.io": 200, "jon@acme.io": 200}},
		{"POST", "/api/v1/products", product, map[string]int{"mark@acme.io": 422, "yuki@acme.io": 422, "priya@acme.io": 403, "jon@acme.io": 403}},
		{"PATCH", "/api/v1/orders/10010/status", map[string]string{"status": "shipped"}, map[string]int{"yuki@acme.io": 200, "priya@acme.io": 200, "jon@acme.io": 403}},
		{"POST", "/api/v1/customers/1/notes", map[string]string{"text": "hi"}, map[string]int{"priya@acme.io": 201, "yuki@acme.io": 403, "jon@acme.io": 403}},
		{"GET", "/api/v1/team", nil, map[string]int{"mark@acme.io": 200, "yuki@acme.io": 200, "priya@acme.io": 403, "jon@acme.io": 403}},
		{"GET", "/api/v1/requests", nil, map[string]int{"artur@acme.io": 200, "mark@acme.io": 200, "yuki@acme.io": 403, "jon@acme.io": 403}},
		{"DELETE", "/api/v1/team/1", nil, map[string]int{"mark@acme.io": 403}}, // owner is protected
	}
	clients := map[string]*client{}
	for _, tc := range cases {
		for email, want := range tc.want {
			c, ok := clients[email]
			if !ok {
				c = newClient(t, srv)
				c.login(email)
				clients[email] = c
			}
			if got, body := c.do(tc.method, tc.path, tc.body); got != want {
				t.Errorf("%s %s as %s: got %d, want %d (%v)", tc.method, tc.path, email, got, want, body)
			}
		}
	}
}

func TestCrossOriginWriteRejected(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("artur@acme.io")
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/v1/products/1", nil)
	req.Header.Set("Origin", "https://evil.example")
	res, err := c.c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 for cross-origin write, got %d", res.StatusCode)
	}
}

func TestProductImages(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("mark@acme.io")

	upload := func(name string, data []byte) (int, map[string]any) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		fw.Write(data)
		mw.Close()
		req, _ := http.NewRequest("POST", srv.URL+"/api/v1/products/1/images", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		res, err := c.c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	if code, _ := upload("evil.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)); code != http.StatusUnsupportedMediaType {
		t.Fatalf("svg upload: want 415, got %d", code)
	}
	code, p := upload("pixel.png", tinyPNG)
	if code != http.StatusCreated {
		t.Fatalf("png upload: %d %v", code, p)
	}
	imgs := p["images"].([]any)
	last := imgs[len(imgs)-1].(map[string]any)
	url, id := last["url"].(string), last["id"].(string)

	res, err := http.Get(srv.URL + url)
	if err != nil || res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("serve uploaded file: %v %v", err, res)
	}
	res.Body.Close()

	if code, p := c.do("POST", "/api/v1/products/1/images/"+id+"/primary", nil); code != 200 || p["images"].([]any)[0].(map[string]any)["id"] != id {
		t.Fatalf("set primary: %d", code)
	}
	if code, _ := c.do("DELETE", "/api/v1/products/1/images/"+id, nil); code != 200 {
		t.Fatalf("delete image: %d", code)
	}
	res, _ = http.Get(srv.URL + url)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("file should be gone after delete, got %d", res.StatusCode)
	}
}

func TestCustomerDetailAndRequestLog(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("artur@acme.io")

	code, det := c.do("GET", "/api/v1/customers/1", nil)
	if code != 200 {
		t.Fatalf("customer detail: %d", code)
	}
	orders := det["orders"].([]any)
	stats := det["stats"].(map[string]any)
	if len(orders) == 0 || stats["orders"].(float64) == 0 {
		t.Fatalf("customer should have a purchase history: %v", stats)
	}

	c.do("PUT", "/api/v1/products/2", map[string]any{"name": "", "password": "secret"}) // 400 unknown field
	code, list := c.do("GET", "/api/v1/requests?method=PUT", nil)
	if code != 200 {
		t.Fatalf("requests: %d", code)
	}
	items := list["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected the PUT to be logged")
	}
	first := items[0].(map[string]any)
	if first["actor"] != "artur@acme.io" {
		t.Fatalf("actor not recorded: %v", first)
	}
	_, entry := c.do("GET", "/api/v1/requests/"+first["id"].(string), nil)
	if body := entry["requestBody"].(string); strings.Contains(body, "secret") {
		t.Fatalf("password leaked into request log: %s", body)
	}
	if entry["requestHeaders"].(map[string]any)["Cookie"] != "[redacted]" {
		t.Fatalf("cookie header not redacted: %v", entry["requestHeaders"])
	}
}
