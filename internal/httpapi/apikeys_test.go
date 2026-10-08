package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// bearer makes a cookie-less request authenticated by an API key.
func bearer(t *testing.T, base, method, path, key string, body any) (int, map[string]any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, base+path, rd)
	req.Header.Set("Authorization", "Bearer "+key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestAPIKeys(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")

	// The seeded demo keys are listed, masked, and never carry a secret.
	code, list := admin.do("GET", "/api/v1/settings/api-keys", nil)
	if code != http.StatusOK || len(list["items"].([]any)) != 3 {
		t.Fatalf("list: %d %v", code, list)
	}
	first := list["items"].([]any)[0].(map[string]any)
	if !strings.HasPrefix(first["masked"].(string), "ga_live_••••••••") || first["secret"] != nil || first["Hash"] != nil {
		t.Fatalf("listed key = %v", first)
	}

	// Create a read key: the secret is returned once.
	code, made := admin.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "  Reporting ", "scope": "read"})
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, made)
	}
	secret := made["secret"].(string)
	key := made["key"].(map[string]any)
	if !strings.HasPrefix(secret, "ga_live_") || key["name"] != "Reporting" || !strings.HasSuffix(secret, key["last4"].(string)) {
		t.Fatalf("created = %v", made)
	}

	// It works without a cookie, for reads only.
	if code, _ := bearer(t, srv.URL, "GET", "/api/v1/orders?limit=1", secret, nil); code != http.StatusOK {
		t.Fatalf("read with key: %d", code)
	}
	for _, path := range []string{"/api/v1/team", "/api/v1/settings/api-keys", "/api/v1/requests"} {
		if code, _ := bearer(t, srv.URL, "GET", path, secret, nil); code != http.StatusForbidden {
			t.Fatalf("read key on %s: got %d, want 403", path, code)
		}
	}
	if code, _ := bearer(t, srv.URL, "PATCH", "/api/v1/orders/10250/status", secret, map[string]any{"status": "shipped"}); code != http.StatusForbidden {
		t.Fatalf("read key writing: got %d, want 403", code)
	}
	if code, _ := bearer(t, srv.URL, "GET", "/api/v1/orders", "ga_live_nope", nil); code != http.StatusUnauthorized {
		t.Fatalf("unknown key: got %d, want 401", code)
	}

	// A write key can change orders, but still not manage keys.
	_, wmade := admin.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "Sync", "scope": "write"})
	wsecret := wmade["secret"].(string)
	_, o := admin.do("GET", "/api/v1/orders?status=paid&limit=1", nil)
	id := o["items"].([]any)[0].(map[string]any)["id"]
	if code, _ := bearer(t, srv.URL, "PATCH", fmt.Sprintf("/api/v1/orders/%v/status", id), wsecret, map[string]any{"status": "shipped"}); code != http.StatusOK {
		t.Fatalf("write key writing: got %d, want 200", code)
	}
	if code, _ := bearer(t, srv.URL, "POST", "/api/v1/settings/api-keys", wsecret, map[string]any{"name": "Evil"}); code != http.StatusForbidden {
		t.Fatalf("key creating keys: got %d, want 403", code)
	}

	// Use is visible in the list; the change is in the audit log under the key's name.
	_, list = admin.do("GET", "/api/v1/settings/api-keys", nil)
	for _, it := range list["items"].([]any) {
		if it := it.(map[string]any); it["name"] == "Reporting" && it["lastUsedAt"] == nil {
			t.Fatal("last use was not recorded")
		}
	}
	_, act := admin.do("GET", "/api/v1/activity?kind=order&limit=1", nil)
	if a := act["items"].([]any)[0].(map[string]any); a["actor"] != "API key “Sync”" {
		t.Fatalf("activity actor = %v", a["actor"])
	}

	// Revoking cuts access at once and removes the key from the list.
	if code, _ := admin.do("DELETE", fmt.Sprintf("/api/v1/settings/api-keys/%v", key["id"]), nil); code != http.StatusNoContent {
		t.Fatalf("revoke: %d", code)
	}
	if code, _ := bearer(t, srv.URL, "GET", "/api/v1/orders", secret, nil); code != http.StatusUnauthorized {
		t.Fatalf("revoked key: got %d, want 401", code)
	}
	if code, _ := admin.do("DELETE", fmt.Sprintf("/api/v1/settings/api-keys/%v", key["id"]), nil); code != http.StatusNotFound {
		t.Fatalf("revoke twice: got %d, want 404", code)
	}

	// Validation, and who may manage keys.
	if code, _ := admin.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "", "scope": "read"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("empty name: got %d, want 422", code)
	}
	if code, _ := admin.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "x", "scope": "root"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("bad scope: got %d, want 422", code)
	}
	editor := newClient(t, srv)
	editor.login("yuki@acme.io")
	if code, _ := editor.do("GET", "/api/v1/settings/api-keys", nil); code != http.StatusForbidden {
		t.Fatalf("editor lists keys: got %d, want 403", code)
	}
}
