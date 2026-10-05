package httpapi_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func perms(t *testing.T, me map[string]any) []string {
	t.Helper()
	var out []string
	for _, p := range me["permissions"].([]any) {
		out = append(out, p.(string))
	}
	return out
}

// The owner grants and revokes individual permissions; the API enforces them.
func TestMemberAccessOverrides(t *testing.T) {
	srv := newServer(t)
	owner := newClient(t, srv)
	owner.login("artur@acme.io")
	jon := newClient(t, srv) // viewer, id 4
	jon.login("jon@acme.io")

	if code, _ := jon.do("GET", "/api/v1/customers", nil); code != http.StatusOK {
		t.Fatalf("viewer reads customers: %d", code)
	}
	code, det := owner.do("PUT", "/api/v1/team/4/access", map[string]any{
		"granted": []string{"orders:write", "dashboard:read"}, // dashboard:read is already in the role: dropped
		"revoked": []string{"customers:read"},
	})
	if code != http.StatusOK {
		t.Fatalf("set access: %d %v", code, det)
	}
	m := det["member"].(map[string]any)
	if g := m["granted"].([]any); len(g) != 1 || g[0] != "orders:write" {
		t.Fatalf("granted = %v, want only orders:write", g)
	}

	_, me := jon.do("GET", "/api/v1/auth/me", nil)
	if p := perms(t, me); !slices.Contains(p, "orders:write") || slices.Contains(p, "customers:read") {
		t.Fatalf("effective permissions = %v", p)
	}
	if code, _ := jon.do("GET", "/api/v1/customers", nil); code != http.StatusForbidden {
		t.Fatalf("revoked customers:read: got %d, want 403", code)
	}
	_, list := jon.do("GET", "/api/v1/orders?status=paid&limit=1", nil)
	id := list["items"].([]any)[0].(map[string]any)["id"]
	if code, _ := jon.do("PATCH", fmt.Sprintf("/api/v1/orders/%v/status", id), map[string]any{"status": "shipped"}); code != http.StatusOK {
		t.Fatalf("granted orders:write: got %d, want 200", code)
	}

	// Only the owner manages exceptions, the danger zone can't be granted and the owner is fixed.
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	if code, _ := admin.do("PUT", "/api/v1/team/4/access", map[string]any{"granted": []string{}, "revoked": []string{}}); code != http.StatusForbidden {
		t.Fatalf("admin sets access: got %d, want 403", code)
	}
	if code, _ := owner.do("PUT", "/api/v1/team/4/access", map[string]any{"granted": []string{"workspace:manage"}, "revoked": []string{}}); code != http.StatusUnprocessableEntity {
		t.Fatalf("grant workspace:manage: got %d, want 422", code)
	}
	if code, _ := owner.do("PUT", "/api/v1/team/1/access", map[string]any{"granted": []string{}, "revoked": []string{"team:read"}}); code != http.StatusForbidden {
		t.Fatalf("change owner access: got %d, want 403", code)
	}

	// Detail shows presence: Jon just made requests.
	_, d := owner.do("GET", "/api/v1/team/4", nil)
	if d["online"] != true || !slices.Contains(perms(t, d), "orders:write") {
		t.Fatalf("member detail = %v", d)
	}

	// The change is in the audit log.
	_, act := owner.do("GET", "/api/v1/activity?kind=role&actor=1&limit=1", nil)
	msg := act["items"].([]any)[0].(map[string]any)["message"]
	if msg != "changed Jon Berg's access: granted Manage orders; revoked View customers" {
		t.Fatalf("activity = %q", msg)
	}
}

// Suspending signs the member out and blocks sign-in until reactivated.
func TestSuspendMember(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	jon := newClient(t, srv)
	jon.login("jon@acme.io")

	if code, _ := admin.do("PUT", "/api/v1/team/4/status", map[string]any{"status": "suspended"}); code != http.StatusOK {
		t.Fatalf("suspend: %d", code)
	}
	if code, _ := jon.do("GET", "/api/v1/auth/me", nil); code != http.StatusUnauthorized {
		t.Fatalf("suspended session: got %d, want 401", code)
	}
	if code, _ := admin.do("PUT", "/api/v1/team/6/status", map[string]any{"status": "suspended"}); code != http.StatusForbidden {
		t.Fatalf("admin suspends admin: got %d, want 403", code)
	}
	if code, _ := admin.do("PUT", "/api/v1/team/2/status", map[string]any{"status": "suspended"}); code != http.StatusForbidden {
		t.Fatalf("suspend self: got %d, want 403", code)
	}
	if code, _ := admin.do("PUT", "/api/v1/team/3/status", map[string]any{"status": "active"}); code != http.StatusConflict {
		t.Fatalf("activate pending invite: got %d, want 409", code)
	}
	if code, _ := admin.do("PUT", "/api/v1/team/4/status", map[string]any{"status": "active"}); code != http.StatusOK {
		t.Fatalf("reactivate: %d", code)
	}
	jon.login("jon@acme.io")
}
