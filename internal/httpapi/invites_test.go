package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestInvitations(t *testing.T) {
	srv := newServer(t)
	owner := newClient(t, srv)
	owner.login("artur@acme.io")
	owner.do("PATCH", "/api/v1/settings", map[string]any{"publicBaseUrl": "https://admin.acme.io"})

	code, m := owner.do("POST", "/api/v1/team", map[string]any{"name": "New Hire", "email": "new.hire@acme.io", "role": "editor"})
	if code != http.StatusCreated || m["status"] != "invited" {
		t.Fatalf("invite: %d %v", code, m)
	}
	inv := m["invite"].(map[string]any)
	token := inv["token"].(string)
	if !strings.HasPrefix(inv["url"].(string), "https://admin.acme.io/invite/") || !strings.HasSuffix(inv["url"].(string), token) || m["inviteExpiresAt"] == nil {
		t.Fatalf("invite = %v", inv)
	}

	// The invitee sees what they were invited to, without signing in.
	guest := newClient(t, srv)
	code, info := guest.do("GET", "/api/v1/auth/invite/"+token, nil)
	if code != http.StatusOK || info["email"] != "new.hire@acme.io" || info["role"] != "editor" {
		t.Fatalf("details: %d %v", code, info)
	}
	if code, _ := guest.do("GET", "/api/v1/auth/invite/not-a-token", nil); code != http.StatusNotFound {
		t.Fatalf("bad token: got %d, want 404", code)
	}
	if code, _ := guest.do("POST", "/api/v1/auth/login", map[string]string{"email": "new.hire@acme.io", "password": "goadmin"}); code != http.StatusUnauthorized {
		t.Fatalf("an invited member can't sign in yet: got %d", code)
	}

	// Choosing a password.
	for name, pw := range map[string]string{"short": "abc123", "the email": "new.hire@acme.io"} {
		if code, _ := guest.do("POST", "/api/v1/auth/invite/"+token, map[string]string{"password": pw}); code != http.StatusUnprocessableEntity {
			t.Fatalf("%s password: got %d, want 422", name, code)
		}
	}
	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/auth/invite/"+token, strings.NewReader(`{"password":"correct horse battery"}`))
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Content-Type", "application/json")
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin accept: %v %v", err, res)
	}

	code, me := guest.do("POST", "/api/v1/auth/invite/"+token, map[string]string{"name": "Nina Hire", "password": "correct horse battery"})
	user, _ := me["user"].(map[string]any)
	if code != http.StatusOK || user["status"] != "active" || user["name"] != "Nina Hire" || user["role"] != "editor" {
		t.Fatalf("accept: %d %v", code, me)
	}
	// Accepting signs them in, with the role's access.
	if code, _ := guest.do("GET", "/api/v1/products", nil); code != http.StatusOK {
		t.Fatalf("signed in after accepting: %d", code)
	}
	if code, _ := guest.do("GET", "/api/v1/settings", nil); code != http.StatusForbidden {
		t.Fatalf("an editor must not read settings: %d", code)
	}
	// The link is single-use, and the password works for a fresh sign-in.
	if code, _ := guest.do("GET", "/api/v1/auth/invite/"+token, nil); code != http.StatusNotFound {
		t.Fatalf("link reuse: got %d, want 404", code)
	}
	again := newClient(t, srv)
	if code, _ := again.do("POST", "/api/v1/auth/login", map[string]string{"email": "new.hire@acme.io", "password": "correct horse battery"}); code != http.StatusOK {
		t.Fatalf("sign in with the chosen password: %d", code)
	}
	_, act := owner.do("GET", "/api/v1/activity?kind=team&limit=1", nil)
	if msg := act["items"].([]any)[0].(map[string]any)["message"]; msg != "accepted the invitation and joined as editor" {
		t.Fatalf("activity = %v", msg)
	}
}

func TestResendInvite(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")

	// Sofia (id 3) was invited in the demo data and has no link yet.
	code, r := admin.do("POST", "/api/v1/team/3/invite", nil)
	if code != http.StatusOK {
		t.Fatalf("resend: %d %v", code, r)
	}
	first := r["invite"].(map[string]any)["token"].(string)
	_, r = admin.do("POST", "/api/v1/team/3/invite", nil)
	second := r["invite"].(map[string]any)["token"].(string)
	guest := newClient(t, srv)
	if code, _ := guest.do("GET", "/api/v1/auth/invite/"+first, nil); code != http.StatusNotFound {
		t.Fatalf("the old link still works: %d", code)
	}
	if code, _ := guest.do("GET", "/api/v1/auth/invite/"+second, nil); code != http.StatusOK {
		t.Fatalf("the new link: %d", code)
	}

	if code, _ := admin.do("POST", "/api/v1/team/4/invite", nil); code != http.StatusConflict {
		t.Fatalf("an active member: got %d, want 409", code)
	}
	viewer := newClient(t, srv)
	viewer.login("jon@acme.io")
	if code, _ := viewer.do("POST", "/api/v1/team/3/invite", nil); code != http.StatusForbidden {
		t.Fatalf("viewer: got %d, want 403", code)
	}
}
