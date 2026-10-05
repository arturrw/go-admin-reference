package httpapi_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func (c *client) raw(method, path, contentType string, body io.Reader) (int, http.Header, []byte) {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.srv.URL+path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, b
}

func readCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	recs, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF")))).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return recs
}

func TestExports(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("jon@acme.io") // viewer: read-only, may still export

	for _, tc := range []struct{ path, first string }{
		{"/api/v1/products/export", "id,name,sku"},
		{"/api/v1/orders/export?status=paid", "id,placed_at,status"},
		{"/api/v1/customers/export", "id,name,email"},
	} {
		code, h, body := c.raw("GET", tc.path, "", nil)
		if code != 200 || !strings.HasPrefix(h.Get("Content-Type"), "text/csv") || !strings.Contains(h.Get("Content-Disposition"), "attachment") {
			t.Fatalf("%s: %d %v", tc.path, code, h)
		}
		recs := readCSV(t, body)
		if len(recs) < 2 || !strings.HasPrefix(strings.Join(recs[0], ","), tc.first) {
			t.Fatalf("%s: unexpected csv %v", tc.path, recs[:min(2, len(recs))])
		}
		if strings.Contains(tc.path, "status=paid") {
			for _, r := range recs[1:] {
				if r[2] != "paid" {
					t.Fatalf("orders export ignored the status filter: %v", r)
				}
			}
		}
	}
}

func TestImportProducts(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("yuki@acme.io") // editor

	// Round-trip: the export is a valid import that changes nothing but the edited row.
	_, _, body := c.raw("GET", "/api/v1/products/export", "", nil)
	recs := readCSV(t, body)
	header := recs[0]
	col := map[string]int{}
	for i, h := range header {
		col[h] = i
	}
	edited := recs[1]
	edited[col["price"]] = "$1,234.50"
	edited[col["stock"]] = "7"

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write(header)
	w.Write(edited)
	w.Write([]string{"", "Test Lamp", "new-001", "lighting", "Acme", "led; desk", "19.9", "", "5", "3", "200", "active", "", "", "Imported", ""})
	w.Flush()

	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	fw, _ := mw.CreateFormFile("file", "products.csv")
	fw.Write(append([]byte("\xEF\xBB\xBF"), buf.Bytes()...))
	mw.Close()
	code, _, out := c.raw("POST", "/api/v1/products/import", mw.FormDataContentType(), &form)
	if code != 200 {
		t.Fatalf("import: %d %s", code, out)
	}
	var res struct{ Created, Updated int }
	json.Unmarshal(out, &res)
	if res.Created != 1 || res.Updated != 1 {
		t.Fatalf("want 1 created + 1 updated, got %+v", res)
	}

	code, p := c.do("GET", "/api/v1/products/"+edited[col["id"]], nil)
	if code != 200 || p["priceCents"].(float64) != 123450 || p["stock"].(float64) != 7 {
		t.Fatalf("updated product: %d price=%v stock=%v", code, p["priceCents"], p["stock"])
	}
	_, list := c.do("GET", "/api/v1/products?q=NEW-001", nil)
	items := list["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["category"] != "Lighting" || items[0].(map[string]any)["priceCents"].(float64) != 1990 {
		t.Fatalf("created product: %v", items)
	}

	// An invalid row rejects the whole file.
	bad := "sku,name,category,price\nGOOD-1,Fine,Audio,10\nBAD-1,,Nope,abc\n"
	code, _, out = c.raw("POST", "/api/v1/products/import", "text/csv", strings.NewReader(bad))
	if code != http.StatusUnprocessableEntity || !strings.Contains(string(out), `"row":3`) {
		t.Fatalf("invalid import: %d %s", code, out)
	}
	if _, list := c.do("GET", "/api/v1/products?q=GOOD-1", nil); len(list["items"].([]any)) != 0 {
		t.Fatal("a rejected import must not write any row")
	}

	// Viewers cannot import.
	v := newClient(t, srv)
	v.login("jon@acme.io")
	if code, _, _ := v.raw("POST", "/api/v1/products/import", "text/csv", strings.NewReader(bad)); code != http.StatusForbidden {
		t.Fatalf("viewer import: want 403, got %d", code)
	}
}

func TestLogLevel(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("jon@acme.io")
	if code, body := c.do("GET", "/api/v1/settings/log-level", nil); code != 200 || body["level"] != "info" {
		t.Fatalf("get level: %d %v", code, body)
	}
	if code, _ := c.do("PUT", "/api/v1/settings/log-level", map[string]string{"level": "debug"}); code != http.StatusForbidden {
		t.Fatalf("viewer set level: want 403, got %d", code)
	}

	a := newClient(t, srv)
	a.login("mark@acme.io")
	if code, body := a.do("PUT", "/api/v1/settings/log-level", map[string]string{"level": "WARN"}); code != 200 || body["level"] != "warn" {
		t.Fatalf("set level: %d %v", code, body)
	}
	if _, body := c.do("GET", "/api/v1/settings/log-level", nil); body["level"] != "warn" {
		t.Fatalf("level did not stick: %v", body)
	}
	if code, _ := a.do("PUT", "/api/v1/settings/log-level", map[string]string{"level": "loud"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("bad level: want 422, got %d", code)
	}
}

func TestLive(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("jon@acme.io")
	code, a := c.do("GET", "/api/v1/live", nil)
	if code != 200 || len(a["history"].([]any)) != 60 || a["hot"] == nil {
		t.Fatalf("live: %d %v", code, a)
	}
	hot := a["hot"].(map[string]any)
	if hot["name"] == "" || hot["viewers"].(float64) <= 0 {
		t.Fatalf("hot product: %v", hot)
	}
	// History is a function of time, so overlapping polls agree.
	_, b := c.do("GET", "/api/v1/live", nil)
	ha, hb := a["history"].([]any), b["history"].([]any)
	if ha[len(ha)-2].(map[string]any)["at"] != hb[len(hb)-2].(map[string]any)["at"] {
		return // crossed a 2s boundary between polls; nothing to compare
	}
	if ha[10].(map[string]any)["requestsPerSec"] != hb[10].(map[string]any)["requestsPerSec"] {
		t.Fatal("history changed between polls")
	}
}
