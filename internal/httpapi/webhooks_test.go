package httpapi_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type received struct {
	header http.Header
	body   []byte
}

// receiver is a webhook endpoint; its first failTests deliveries of test
// events answer 500 (other events always succeed).
func receiver(t *testing.T, failTests int32) (*httptest.Server, chan received) {
	t.Helper()
	got := make(chan received, 20)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Header.Get("X-GoAdmin-Event") == "webhook.test" && calls.Add(1) <= failTests {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		got <- received{r.Header.Clone(), b}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

// next waits for a delivered event of the given type, skipping others.
func next(t *testing.T, ch chan received, typ string) received {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case r := <-ch:
			if r.header.Get("X-GoAdmin-Event") == typ {
				return r
			}
		case <-timeout:
			t.Fatalf("no %s webhook arrived", typ)
			return received{}
		}
	}
}

func TestWebhooks(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")
	hook, got := receiver(t, 0)

	// Bad URLs are refused.
	for _, u := range []string{"admin.acme.io/hook", "ftp://x.io/hook", "http://user:pw@x.io/hook"} {
		if code, _ := admin.do("PATCH", "/api/v1/settings", map[string]any{"webhookUrl": u}); code != http.StatusUnprocessableEntity {
			t.Fatalf("%q: got %d, want 422", u, code)
		}
	}
	if code, _ := admin.do("POST", "/api/v1/settings/webhook/test", nil); code != http.StatusConflict {
		t.Fatalf("test without a URL: got %d, want 409", code)
	}

	// Turning signing on creates the secret.
	code, s := admin.do("PATCH", "/api/v1/settings", map[string]any{"webhookUrl": hook.URL, "webhooksSigned": true, "publicBaseUrl": "https://admin.acme.io"})
	secret, _ := s["webhookSecret"].(string)
	if code != http.StatusOK || !strings.HasPrefix(secret, "whsec_") {
		t.Fatalf("enable: %d %v", code, s)
	}

	// A change in the admin arrives as a signed event.
	admin.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "Hello hook"})
	r := next(t, got, "customer.note")
	if r.header.Get("Content-Type") != "application/json" {
		t.Fatalf("headers = %v", r.header)
	}
	ts := r.header.Get("X-GoAdmin-Timestamp")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(r.body)))
	if want := "sha256=" + hex.EncodeToString(mac.Sum(nil)); r.header.Get("X-GoAdmin-Signature") != want {
		t.Fatalf("signature = %q, want %q", r.header.Get("X-GoAdmin-Signature"), want)
	}
	var ev struct {
		ID, Type, Message, URL string
		Actor                  struct{ Name string }
		Entity                 struct {
			Type string
			ID   int
		}
	}
	if err := json.Unmarshal(r.body, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Type != "customer.note" || ev.Actor.Name != "Mark Liu" || ev.Entity.Type != "customer" || ev.Entity.ID != 4 ||
		ev.URL != "https://admin.acme.io/customers?view=4" || !strings.HasPrefix(ev.Message, "added a note on ") || r.header.Get("X-GoAdmin-Delivery") != ev.ID {
		t.Fatalf("event = %+v", ev)
	}

	// Sign-ins are not events. With signing off the signature header is gone.
	admin.login("mark@acme.io")
	admin.do("PATCH", "/api/v1/settings", map[string]any{"webhooksSigned": false})
	// Deliveries are asynchronous, so the earlier settings event may still be in flight.
	for r := next(t, got, "settings.changed"); !strings.Contains(string(r.body), "turned webhook signing off"); r = next(t, got, "settings.changed") {
	}
	admin.do("POST", "/api/v1/customers/4/notes", map[string]any{"text": "Unsigned"})
	r = next(t, got, "customer.note")
	if r.header.Get("X-GoAdmin-Signature") != "" {
		t.Fatalf("unsigned event carries a signature: %v", r.header)
	}

	// The test button, and the recent deliveries it shows.
	code, dl := admin.do("POST", "/api/v1/settings/webhook/test", nil)
	if code != http.StatusOK || dl["ok"] != true || dl["status"].(float64) != 204 || dl["type"] != "webhook.test" {
		t.Fatalf("test delivery: %d %v", code, dl)
	}
	next(t, got, "webhook.test")
	_, list := admin.do("GET", "/api/v1/settings/webhook/deliveries", nil)
	if items := list["items"].([]any); len(items) < 3 || items[0].(map[string]any)["type"] != "webhook.test" {
		t.Fatalf("deliveries = %v", list)
	}

	// Rotating the secret changes it, and is logged without revealing it.
	_, rot := admin.do("POST", "/api/v1/settings/webhook/rotate-secret", nil)
	if rot["webhookSecret"] == secret || !strings.HasPrefix(rot["webhookSecret"].(string), "whsec_") {
		t.Fatalf("rotate = %v", rot)
	}
	_, act := admin.do("GET", "/api/v1/activity?kind=settings&limit=1", nil)
	if m := act["items"].([]any)[0].(map[string]any)["message"].(string); m != "rotated the webhook signing secret" {
		t.Fatalf("activity = %q", m)
	}

	viewer := newClient(t, srv)
	viewer.login("jon@acme.io")
	if code, _ := viewer.do("POST", "/api/v1/settings/webhook/test", nil); code != http.StatusForbidden {
		t.Fatalf("viewer: got %d, want 403", code)
	}
}

// A receiver that fails is retried; one that stays down is reported.
func TestWebhookRetries(t *testing.T) {
	srv := newServer(t)
	admin := newClient(t, srv)
	admin.login("mark@acme.io")

	flaky, got := receiver(t, 2) // fails the test event twice, then works
	admin.do("PATCH", "/api/v1/settings", map[string]any{"webhookUrl": flaky.URL})
	_, dl := admin.do("POST", "/api/v1/settings/webhook/test", nil)
	if dl["ok"] != true || dl["attempts"].(float64) != 3 {
		t.Fatalf("flaky receiver: %v", dl)
	}
	next(t, got, "webhook.test")

	down, _ := receiver(t, 100)
	admin.do("PATCH", "/api/v1/settings", map[string]any{"webhookUrl": down.URL})
	_, dl = admin.do("POST", "/api/v1/settings/webhook/test", nil)
	if dl["ok"] != false || dl["attempts"].(float64) != 3 || dl["status"].(float64) != 500 || !strings.Contains(dl["error"].(string), "500") {
		t.Fatalf("dead receiver: %v", dl)
	}
}
