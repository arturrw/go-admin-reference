package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
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

// A refund needs a reason, which stays on the order and in the customer's history.
func TestRefundKeepsReason(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("priya@acme.io")

	_, list := c.do("GET", "/api/v1/orders?status=delivered&limit=1", nil)
	o := list["items"].([]any)[0].(map[string]any)
	path := fmt.Sprintf("/api/v1/orders/%v/status", o["id"])

	if code, res := c.do("PATCH", path, map[string]any{"status": "refunded"}); code != http.StatusUnprocessableEntity || res["fields"].(map[string]any)["reason"] == nil {
		t.Fatalf("refund without reason: %d %v", code, res)
	}
	code, got := c.do("PATCH", path, map[string]any{"status": "refunded", "reason": "  Damaged in transit "})
	if code != http.StatusOK {
		t.Fatalf("refund: %d %v", code, got)
	}
	refund, _ := got["refund"].(map[string]any)
	if got["status"] != "refunded" || refund["reason"] != "Damaged in transit" || refund["by"] != "Priya Shah" {
		t.Fatalf("refunded order = %v", got)
	}
	if code, _ := c.do("PATCH", path, map[string]any{"status": "refunded", "reason": "again"}); code != http.StatusConflict {
		t.Fatalf("second refund: got %d, want 409", code)
	}

	cust := o["customer"].(map[string]any)["id"]
	_, det := c.do("GET", fmt.Sprintf("/api/v1/customers/%v", cust), nil)
	found := false
	for _, x := range det["orders"].([]any) {
		if x := x.(map[string]any); x["id"] == o["id"] {
			found = x["refund"].(map[string]any)["reason"] == "Damaged in transit"
		}
	}
	if !found {
		t.Fatal("customer history does not show the refund reason")
	}

	_, act := c.do("GET", "/api/v1/dashboard", nil)
	first := act["activity"].([]any)[0].(map[string]any)
	if first["kind"] != "refund" || !strings.HasSuffix(first["message"].(string), "— Damaged in transit") {
		t.Fatalf("activity = %v", first)
	}
}

// from/to restrict orders to a time window, as the dashboard's day view uses.
func TestOrdersByDateRange(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("mark@acme.io")

	_, all := c.do("GET", "/api/v1/orders?limit=100", nil)
	items := all["items"].([]any)
	newest := items[0].(map[string]any)["placedAt"].(string)
	at, _ := time.Parse(time.RFC3339, newest)
	from, to := at.Add(-6*time.Hour), at.Add(time.Second)

	want := 0
	for _, it := range items {
		p, _ := time.Parse(time.RFC3339, it.(map[string]any)["placedAt"].(string))
		if !p.Before(from) && p.Before(to) {
			want++
		}
	}
	q := url.Values{"from": {from.Format(time.RFC3339)}, "to": {to.Format(time.RFC3339)}, "limit": {"100"}}
	code, got := c.do("GET", "/api/v1/orders?"+q.Encode(), nil)
	if code != 200 || int(got["total"].(float64)) != want || want == 0 {
		t.Fatalf("orders in window: code %d, total %v, want %d", code, got["total"], want)
	}
	if code, _ := c.do("GET", "/api/v1/orders?from=yesterday", nil); code != http.StatusBadRequest {
		t.Fatalf("bad from: got %d, want 400", code)
	}
}

func rawGet(t *testing.T, c *client, path string) (int, http.Header, string) {
	t.Helper()
	res, err := c.c.Get(c.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header, string(b)
}

// The invoice is a printable page whose numbers match the order sheet's.
func TestInvoice(t *testing.T) {
	srv := newServer(t)
	viewer := newClient(t, srv)
	viewer.login("jon@acme.io")

	_, list := viewer.do("GET", "/api/v1/orders?status=delivered&limit=1", nil)
	id := fmt.Sprint(list["items"].([]any)[0].(map[string]any)["id"])
	_, o := viewer.do("GET", "/api/v1/orders/"+id, nil)
	total, ship, tax, grand := int64(o["totalCents"].(float64)), int64(o["shippingCents"].(float64)), int64(o["taxCents"].(float64)), int64(o["grandTotalCents"].(float64))
	if grand != total+ship+tax || tax != (total*8+50)/100 || (total > 10000) != (ship == 0) {
		t.Fatalf("totals: goods %d shipping %d tax %d grand %d", total, ship, tax, grand)
	}

	code, h, html := rawGet(t, viewer, "/api/v1/orders/"+id+"/invoice")
	cust := o["customer"].(map[string]any)["name"].(string)
	if code != http.StatusOK || !strings.HasPrefix(h.Get("Content-Type"), "text/html") {
		t.Fatalf("invoice: %d %v", code, h)
	}
	for _, want := range []string{"INV-" + id, cust, "Print / Save as PDF", usd(grand), usd(tax)} {
		if !strings.Contains(html, want) {
			t.Errorf("invoice lacks %q", want)
		}
	}
	for _, it := range o["items"].([]any) {
		if sku := it.(map[string]any)["sku"].(string); !strings.Contains(html, sku) {
			t.Errorf("invoice lacks item %s", sku)
		}
	}
	if strings.Contains(html, "REFUNDED") {
		t.Error("a delivered order is stamped refunded")
	}

	// A refund shows up on the invoice with its reason.
	support := newClient(t, srv)
	support.login("priya@acme.io")
	support.do("PATCH", "/api/v1/orders/"+id+"/status", map[string]any{"status": "refunded", "reason": "Arrived <b>broken</b>"})
	_, _, html = rawGet(t, viewer, "/api/v1/orders/"+id+"/invoice")
	if !strings.Contains(html, "REFUNDED") || !strings.Contains(html, "Arrived &lt;b&gt;broken&lt;/b&gt;") || strings.Contains(html, "<b>broken</b>") {
		t.Errorf("refund not shown or not escaped")
	}

	if code, _, _ := rawGet(t, viewer, "/api/v1/orders/99999999/invoice"); code != http.StatusNotFound {
		t.Errorf("unknown order: %d", code)
	}
	anon := newClient(t, srv)
	if code, _, _ := rawGet(t, anon, "/api/v1/orders/"+id+"/invoice"); code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", code)
	}
}

// usd formats cents like the invoice does.
func usd(c int64) string {
	whole := fmt.Sprint(c / 100)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("$%s.%02d", whole, c%100)
}
