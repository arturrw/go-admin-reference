package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"
)

// Order items show the product's current cover, not the one at purchase time.
func TestOrderItemsFollowProductCover(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("mark@acme.io")

	_, list := c.do("GET", "/api/v1/orders?limit=1", nil)
	order := list["items"].([]any)[0].(map[string]any)
	item := order["items"].([]any)[0].(map[string]any)
	pid := int64(item["productId"].(float64))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "cover.png")
	fw.Write(tinyPNG)
	mw.Close()
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/products/%d/images", srv.URL, pid), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.c.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("upload: %v %v", err, res)
	}
	var p map[string]any
	json.NewDecoder(res.Body).Decode(&p)
	res.Body.Close()
	imgs := p["images"].([]any)
	up := imgs[len(imgs)-1].(map[string]any)
	if code, _ := c.do("POST", fmt.Sprintf("/api/v1/products/%d/images/%s/primary", pid, up["id"]), nil); code != 200 {
		t.Fatalf("set primary: %d", code)
	}

	cover := func(o map[string]any) string {
		for _, it := range o["items"].([]any) {
			if it := it.(map[string]any); int64(it["productId"].(float64)) == pid {
				return it["imageUrl"].(string)
			}
		}
		t.Fatal("product not in order")
		return ""
	}
	_, o := c.do("GET", fmt.Sprintf("/api/v1/orders/%v", order["id"]), nil)
	if got := cover(o); got != up["url"] {
		t.Fatalf("order detail image = %q, want new cover %q", got, up["url"])
	}
	_, list = c.do("GET", "/api/v1/orders?limit=1", nil)
	if got := cover(list["items"].([]any)[0].(map[string]any)); got != up["url"] {
		t.Fatalf("order list image = %q, want new cover %q", got, up["url"])
	}
}
