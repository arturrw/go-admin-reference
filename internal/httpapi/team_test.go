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

// The owner sets the quarter's goal; the dashboard target and pace follow it.
func TestQuarterTarget(t *testing.T) {
	srv := newServer(t)
	owner := newClient(t, srv)
	owner.login("artur@acme.io")

	_, before := owner.do("GET", "/api/v1/dashboard", nil)
	tg := before["target"].(map[string]any)
	if tg["goalCents"].(float64) != 120_000_000 || tg["updatedBy"] != "" {
		t.Fatalf("default target = %v", tg)
	}

	code, after := owner.do("PUT", "/api/v1/target", map[string]any{"goalCents": 60_000_000})
	if code != http.StatusOK || after["goalCents"].(float64) != 60_000_000 || after["updatedBy"] != "Artur DCS" {
		t.Fatalf("set target: %d %v", code, after)
	}
	if after["pacePct"].(float64) <= tg["pacePct"].(float64) {
		t.Fatalf("halving the goal should improve the pace: %v → %v", tg["pacePct"], after["pacePct"])
	}
	_, dash := owner.do("GET", "/api/v1/dashboard", nil)
	if dash["target"].(map[string]any)["goalCents"].(float64) != 60_000_000 {
		t.Fatalf("dashboard target = %v", dash["target"])
	}
	_, act := owner.do("GET", "/api/v1/activity?kind=target&limit=1", nil)
	if msg := act["items"].([]any)[0].(map[string]any)["message"].(string); msg != "set the "+tg["label"].(string)[:2]+" "+tg["quarter"].(string)[:4]+" target to $600,000.00 (was $1,200,000.00)" {
		t.Fatalf("activity = %q", msg)
	}

	if code, _ := owner.do("PUT", "/api/v1/target", map[string]any{"goalCents": 0}); code != http.StatusUnprocessableEntity {
		t.Fatalf("zero goal: got %d, want 422", code)
	}
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	if code, _ := admin.do("PUT", "/api/v1/target", map[string]any{"goalCents": 1}); code != http.StatusForbidden {
		t.Fatalf("admin sets target: got %d, want 403", code)
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
