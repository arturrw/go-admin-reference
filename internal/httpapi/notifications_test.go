package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func notes(t *testing.T, c *client) (items []map[string]any, unread int) {
	t.Helper()
	code, n := c.do("GET", "/api/v1/notifications", nil)
	if code != http.StatusOK {
		t.Fatalf("notifications: %d", code)
	}
	for _, it := range n["items"].([]any) {
		items = append(items, it.(map[string]any))
	}
	return items, int(n["unread"].(float64))
}

func TestNotifications(t *testing.T) {
	srv := newServer(t)
	mark := newClient(t, srv)
	mark.login("mark@acme.io")
	jon := newClient(t, srv)
	jon.login("jon@acme.io")
	priya := newClient(t, srv)
	priya.login("priya@acme.io")

	// Never opened: the last day of activity from others is unread, own actions and sign-ins are not.
	items, unread := notes(t, jon)
	if unread == 0 || len(items) == 0 {
		t.Fatalf("a fresh member should see recent activity: %d unread of %d", unread, len(items))
	}
	for _, it := range items {
		if it["kind"] == "auth" {
			t.Fatal("sign-ins are not notifications")
		}
	}

	if code, _ := jon.do("POST", "/api/v1/notifications/read", nil); code != http.StatusNoContent {
		t.Fatalf("mark read: %d", code)
	}
	if _, unread := notes(t, jon); unread != 0 {
		t.Fatalf("after reading: %d unread", unread)
	}

	// Someone else's action arrives as unread; the actor isn't notified of their own.
	time.Sleep(20 * time.Millisecond)
	mark.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "ping"})
	items, unread = notes(t, jon)
	if unread != 1 || !strings.HasPrefix(items[0]["message"].(string), "added a note on ") || items[0]["unread"] != true {
		t.Fatalf("jon: %d unread, newest %v", unread, items[0])
	}
	if own, _ := notes(t, mark); strings.HasPrefix(own[0]["message"].(string), "added a note on ") && own[0]["actor"] == "Mark Liu" {
		t.Fatal("mark was notified of his own action")
	}

	// Security alerts reach only the member they are about.
	signInAs(t, srv.URL, "jon@acme.io", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:127.0) Gecko/20100101 Firefox/127.0")
	jonItems, _ := notes(t, jon)
	if jonItems[0]["kind"] != "alert" || !strings.Contains(jonItems[0]["message"].(string), "Firefox on macOS") {
		t.Fatalf("jon's newest = %v", jonItems[0])
	}
	for _, it := range func() []map[string]any { i, _ := notes(t, priya); return i }() {
		if it["kind"] == "alert" && strings.Contains(it["message"].(string), "Jon Berg") {
			t.Fatal("priya sees jon's security alert")
		}
	}

	// API keys have no inbox.
	_, made := mark.do("POST", "/api/v1/settings/api-keys", map[string]any{"name": "K", "scope": "read"})
	if code, n := bearer(t, srv.URL, "GET", "/api/v1/notifications", made["secret"].(string), nil); code != http.StatusOK || n["unread"].(float64) != 0 {
		t.Fatalf("key inbox: %d %v", code, n)
	}
}
