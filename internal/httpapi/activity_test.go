package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"
)

// Changes made through the API land in the audit log and the dashboard feed.
func TestActivityRecordsChanges(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("mark@acme.io")

	_, list := c.do("GET", "/api/v1/products?status=draft", nil)
	p := list["items"].([]any)[0].(map[string]any)
	in := map[string]any{}
	for _, k := range []string{"name", "sku", "category", "vendor", "tags", "priceCents", "compareAtCents", "costCents", "stock", "weightGrams", "description"} {
		in[k] = p[k]
	}
	in["status"] = "active"
	if code, _ := c.do("PUT", fmt.Sprintf("/api/v1/products/%v", p["id"]), in); code != http.StatusOK {
		t.Fatalf("publish: %d", code)
	}

	code, act := c.do("GET", "/api/v1/activity?limit=5", nil)
	if code != http.StatusOK {
		t.Fatalf("activity: %d", code)
	}
	first := act["items"].([]any)[0].(map[string]any)
	want := "published " + p["name"].(string)
	if first["kind"] != "publish" || first["actor"] != "Mark Liu" || first["message"] != want || first["entity"] != "product" || first["entityId"] != p["id"] {
		t.Fatalf("latest entry = %v, want Mark Liu %q", first, want)
	}

	// Mark's sign-in is logged but kept out of the dashboard feed.
	_, mine := c.do("GET", "/api/v1/activity?kind=auth&actor=2", nil)
	if mine["total"].(float64) < 1 {
		t.Fatal("sign-in was not recorded")
	}
	_, dash := c.do("GET", "/api/v1/dashboard", nil)
	feed := dash["activity"].([]any)
	if feed[0].(map[string]any)["message"] != want {
		t.Fatalf("dashboard feed starts with %v", feed[0])
	}
	for _, a := range feed {
		if a.(map[string]any)["kind"] == "auth" {
			t.Fatal("sign-ins must not appear in the dashboard feed")
		}
	}

	// The full log is for roles that can see the team.
	v := newClient(t, srv)
	v.login("jon@acme.io")
	if code, _ := v.do("GET", "/api/v1/activity", nil); code != http.StatusForbidden {
		t.Fatalf("viewer: got %d, want 403", code)
	}
	if code, _ := c.do("GET", "/api/v1/activity?kind=nope", nil); code != http.StatusBadRequest {
		t.Fatalf("bad kind: got %d, want 400", code)
	}
}
