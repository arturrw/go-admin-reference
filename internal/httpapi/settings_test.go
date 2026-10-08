package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
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

func TestSessionLifetimeSetting(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")

	_, s := admin.do("GET", "/api/v1/settings", nil)
	if s["sessionTtlSeconds"].(float64) <= 0 {
		t.Fatalf("effective lifetime missing: %v", s)
	}
	code, s := admin.do("PATCH", "/api/v1/settings", map[string]any{"sessionTtlSeconds": 8 * 3600})
	if code != http.StatusOK || s["sessionTtlSeconds"].(float64) != 8*3600 {
		t.Fatalf("set lifetime: %d %v", code, s)
	}
	// New sessions use it: the cookie's Max-Age is the lifetime.
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/auth/login", strings.NewReader(`{"email":"jon@acme.io","password":"goadmin"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	var maxAge int
	for _, c := range res.Cookies() {
		maxAge = c.MaxAge
	}
	if maxAge != 8*3600 {
		t.Fatalf("cookie Max-Age = %d, want %d", maxAge, 8*3600)
	}
	if code, _ := admin.do("PATCH", "/api/v1/settings", map[string]any{"sessionTtlSeconds": 100}); code != http.StatusUnprocessableEntity {
		t.Fatalf("odd lifetime: got %d, want 422", code)
	}
	_, act := admin.do("GET", "/api/v1/activity?kind=settings&limit=1", nil)
	if m := act["items"].([]any)[0].(map[string]any)["message"]; m != "set the session lifetime to 8 hours" {
		t.Fatalf("activity = %v", m)
	}
}

// With the audit log off nothing new is recorded, except that switching it is.
func TestAuditLogSwitch(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	total := func() int {
		_, a := admin.do("GET", "/api/v1/activity?limit=1", nil)
		return int(a["total"].(float64))
	}

	n := total()
	admin.do("PATCH", "/api/v1/settings", map[string]any{"auditLog": false})
	if total() != n+1 {
		t.Fatalf("switching the audit log off was not recorded: %d → %d", n, total())
	}
	if code, _ := admin.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "unlogged"}); code != http.StatusCreated {
		t.Fatalf("note: %d", code)
	}
	admin.login("mark@acme.io") // a sign-in is not logged either
	if total() != n+1 {
		t.Fatalf("changes were logged while the audit log was off: %d → %d", n+1, total())
	}
	admin.do("PATCH", "/api/v1/settings", map[string]any{"auditLog": true})
	admin.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "logged"})
	if total() != n+3 {
		t.Fatalf("after switching back on: %d, want %d", total(), n+3)
	}
}

func signInAs(t *testing.T, base, email, userAgent string) {
	t.Helper()
	req, _ := http.NewRequest("POST", base+"/api/v1/auth/login", strings.NewReader(fmt.Sprintf(`{"email":%q,"password":"goadmin"}`, email)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("sign in %s: %v %v", email, err, res)
	}
	res.Body.Close()
}

// A sign-in from a device the member hasn't used before raises an alert for them.
func TestLoginAlerts(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	// Alerts raised for Jon (member 4); the admin's own sign-in raised one for Mark.
	alerts := func() []any {
		_, a := admin.do("GET", "/api/v1/activity?kind=alert&limit=50", nil)
		var mine []any
		for _, e := range a["items"].([]any) {
			if e.(map[string]any)["entityId"].(float64) == 4 {
				mine = append(mine, e)
			}
		}
		return mine
	}
	const (
		win = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
		mac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:127.0) Gecko/20100101 Firefox/127.0"
	)

	signInAs(t, srv.URL, "jon@acme.io", win) // Jon has signed in before (seed), from another device
	a := alerts()
	if len(a) != 1 {
		t.Fatalf("first sign-in from a new device: %d alerts, want 1", len(a))
	}
	got := a[0].(map[string]any)
	if got["entity"] != "member" || got["entityId"].(float64) != 4 || !strings.Contains(got["message"].(string), "Chrome on Windows") {
		t.Fatalf("alert = %v", got)
	}
	signInAs(t, srv.URL, "jon@acme.io", win)
	if len(alerts()) != 1 {
		t.Fatal("the same device alerted twice")
	}
	signInAs(t, srv.URL, "jon@acme.io", mac)
	if len(alerts()) != 2 {
		t.Fatal("a second device did not alert")
	}

	admin.do("PATCH", "/api/v1/settings", map[string]any{"loginAlerts": false})
	signInAs(t, srv.URL, "jon@acme.io", "curl/8.5.0")
	if len(alerts()) != 2 {
		t.Fatal("alerts were raised while switched off")
	}

	// Alerts stay out of the dashboard feed.
	_, dash := admin.do("GET", "/api/v1/dashboard", nil)
	for _, e := range dash["activity"].([]any) {
		if e.(map[string]any)["kind"] == "alert" {
			t.Fatal("alert in the dashboard feed")
		}
	}
}
