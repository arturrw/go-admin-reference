package httpapi_test

import (
	"net/http"
	"testing"
)

func TestWorkspaceSettings(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")

	code, s := admin.do("GET", "/api/v1/settings", nil)
	if code != http.StatusOK || s["serviceName"] != "goadmin-api" || s["publicBaseUrl"] != "" || s["listenAddr"] == nil {
		t.Fatalf("defaults: %d %v", code, s)
	}

	code, s = admin.do("PATCH", "/api/v1/settings", map[string]any{"serviceName": "  Acme Admin ", "publicBaseUrl": "https://admin.acme.io/"})
	if code != http.StatusOK || s["serviceName"] != "Acme Admin" || s["publicBaseUrl"] != "https://admin.acme.io" {
		t.Fatalf("patch: %d %v", code, s)
	}
	// A patch with one field leaves the other alone.
	_, s = admin.do("PATCH", "/api/v1/settings", map[string]any{"serviceName": "Acme"})
	if s["publicBaseUrl"] != "https://admin.acme.io" {
		t.Fatalf("partial patch lost a field: %v", s)
	}

	// Everyone signed in sees the name, in /meta.
	viewer := newClient(t, srv)
	viewer.login("jon@acme.io")
	if _, m := viewer.do("GET", "/api/v1/meta", nil); m["serviceName"] != "Acme" {
		t.Fatalf("meta = %v", m)
	}

	for name, body := range map[string]map[string]any{
		"empty name":   {"serviceName": " "},
		"long name":    {"serviceName": string(make([]byte, 61))},
		"not a URL":    {"publicBaseUrl": "admin.acme.io"},
		"wrong scheme": {"publicBaseUrl": "ftp://admin.acme.io"},
	} {
		if code, _ := admin.do("PATCH", "/api/v1/settings", body); code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: got %d, want 422", name, code)
		}
	}
	if code, _ := admin.do("PATCH", "/api/v1/settings", map[string]any{"publicBaseUrl": ""}); code != http.StatusOK {
		t.Fatalf("clearing the URL: %d", code)
	}

	if code, _ := viewer.do("GET", "/api/v1/settings", nil); code != http.StatusForbidden {
		t.Fatalf("viewer reads settings: got %d, want 403", code)
	}
	if code, _ := viewer.do("PATCH", "/api/v1/settings", map[string]any{"serviceName": "x"}); code != http.StatusForbidden {
		t.Fatalf("viewer patches settings: got %d, want 403", code)
	}

	_, act := admin.do("GET", "/api/v1/activity?kind=settings&limit=3", nil)
	first := act["items"].([]any)[0].(map[string]any)["message"]
	if first != "cleared the public base URL" {
		t.Fatalf("activity = %v", first)
	}
}

// Maintenance mode locks out everyone but owners and admins, API keys included.
func TestMaintenanceMode(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	viewer := newClient(t, srv)
	viewer.login("jon@acme.io")
	_, made := admin.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "Storefront", "scope": "read"})
	key := made["secret"].(string)

	if code, _ := viewer.do("GET", "/api/v1/orders", nil); code != http.StatusOK {
		t.Fatalf("before: %d", code)
	}
	if code, s := admin.do("PATCH", "/api/v1/settings", map[string]any{"maintenance": true}); code != http.StatusOK || s["maintenance"] != true {
		t.Fatalf("turn on: %d %v", code, s)
	}

	// Locked out, but the UI can still ask who it is and why.
	code, body := viewer.do("GET", "/api/v1/orders", nil)
	if code != http.StatusServiceUnavailable || body["maintenance"] != true {
		t.Fatalf("viewer during maintenance: %d %v", code, body)
	}
	if code, _ := bearer(t, srv.URL, "GET", "/api/v1/orders", key, nil); code != http.StatusServiceUnavailable {
		t.Fatalf("API key during maintenance: got %d, want 503", code)
	}
	if code, _ := viewer.do("GET", "/api/v1/auth/me", nil); code != http.StatusOK {
		t.Fatalf("me during maintenance: %d", code)
	}
	if _, m := viewer.do("GET", "/api/v1/meta", nil); m["maintenance"] != true {
		t.Fatalf("meta = %v", m)
	}
	// Owners and admins carry on, and new sign-ins of other roles are refused.
	if code, _ := admin.do("GET", "/api/v1/orders", nil); code != http.StatusOK {
		t.Fatalf("admin during maintenance: %d", code)
	}
	late := newClient(t, srv)
	if code, _ := late.do("POST", "/api/v1/auth/login", map[string]string{"email": "yuki@acme.io", "password": "goadmin"}); code != http.StatusServiceUnavailable {
		t.Fatalf("editor sign-in during maintenance: got %d, want 503", code)
	}
	if code, _ := late.do("POST", "/api/v1/auth/login", map[string]string{"email": "artur@acme.io", "password": "goadmin"}); code != http.StatusOK {
		t.Fatalf("owner sign-in during maintenance: got %d, want 200", code)
	}

	admin.do("PATCH", "/api/v1/settings", map[string]any{"maintenance": false})
	if code, _ := viewer.do("GET", "/api/v1/orders", nil); code != http.StatusOK {
		t.Fatalf("after: %d", code)
	}
	if code, _ := bearer(t, srv.URL, "GET", "/api/v1/orders", key, nil); code != http.StatusOK {
		t.Fatalf("API key after: %d", code)
	}
	_, act := admin.do("GET", "/api/v1/activity?kind=settings&limit=2", nil)
	if m := act["items"].([]any)[0].(map[string]any)["message"]; m != "turned maintenance mode off" {
		t.Fatalf("activity = %v", m)
	}
}
