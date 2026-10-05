package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
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

// Deleting a customer note is allowed to note writers and leaves a trace.
func TestDeleteCustomerNote(t *testing.T) {
	srv := newServer(t)
	c := newClient(t, srv)
	c.login("priya@acme.io")

	code, note := c.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "Temporary note to delete"})
	if code != http.StatusCreated {
		t.Fatalf("add note: %d", code)
	}
	path := fmt.Sprintf("/api/v1/customers/4/notes/%v", note["id"])

	v := newClient(t, srv)
	v.login("jon@acme.io")
	if code, _ := v.do("DELETE", path, nil); code != http.StatusForbidden {
		t.Fatalf("viewer delete: got %d, want 403", code)
	}
	if code, _ := c.do("DELETE", fmt.Sprintf("/api/v1/customers/5/notes/%v", note["id"]), nil); code != http.StatusNotFound {
		t.Fatalf("note of another customer: got %d, want 404", code)
	}
	if code, _ := c.do("DELETE", path, nil); code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", code)
	}
	if code, _ := c.do("DELETE", path, nil); code != http.StatusNotFound {
		t.Fatalf("second delete: got %d, want 404", code)
	}

	_, det := c.do("GET", "/api/v1/customers/4", nil)
	for _, n := range det["customer"].(map[string]any)["notes"].([]any) {
		if n.(map[string]any)["id"] == note["id"] {
			t.Fatal("note still listed")
		}
	}
	_, dash := c.do("GET", "/api/v1/dashboard", nil)
	msg := dash["activity"].([]any)[0].(map[string]any)["message"].(string)
	if !strings.HasPrefix(msg, "deleted Priya Shah's note on ") || !strings.HasSuffix(msg, "“Temporary note to delete”") {
		t.Fatalf("activity message = %q", msg)
	}
}
