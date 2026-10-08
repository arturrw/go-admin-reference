package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// Webhooks send a JSON event to Settings → Webhook URL for every change that
// is written to the activity log (sign-ins and security alerts excluded). The
// request is signed when signing is on:
//
//	X-GoAdmin-Timestamp: 1767225600
//	X-GoAdmin-Signature: sha256=hex(HMAC-SHA256(secret, timestamp + "." + body))
//
// Delivery is asynchronous with up to three attempts; the last results are
// kept in memory (per instance) for the Settings page.

const (
	webhookAttempts = 3
	maxInflight     = 32
	deliveriesKept  = 20
)

type webhookEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Actor     struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"actor"`
	Message string `json:"message"`
	Entity  *struct {
		Type string `json:"type"`
		ID   int64  `json:"id"`
	} `json:"entity,omitempty"`
	// URL opens the record in the admin, when a public base URL is set.
	URL string `json:"url,omitempty"`
}

// Delivery is one webhook event and how sending it went.
type delivery struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	URL        string    `json:"url"`
	OK         bool      `json:"ok"`
	Status     int       `json:"status"` // 0 when the request never got a response
	Error      string    `json:"error,omitempty"`
	Attempts   int       `json:"attempts"`
	DurationMs float64   `json:"durationMs"`
	At         time.Time `json:"at"`
}

type webhookState struct {
	mu       sync.Mutex
	recent   []delivery // newest first
	inflight atomic.Int32
	client   *http.Client
	backoff  []time.Duration // wait before attempt 2, 3
}

func newWebhookState(backoff []time.Duration) *webhookState {
	if len(backoff) == 0 {
		backoff = []time.Duration{time.Second, 5 * time.Second}
	}
	return &webhookState{
		backoff: backoff,
		client: &http.Client{
			Timeout: 5 * time.Second,
			// A receiver answering with a redirect must not steer us elsewhere.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

var eventTypes = map[string]string{
	d.ActProduct: "product.changed", d.ActPublish: "product.published", d.ActImage: "product.images_changed",
	d.ActImport: "products.imported", d.ActOrder: "order.updated", d.ActRefund: "order.refunded",
	d.ActNote: "customer.note", d.ActTeam: "team.changed", d.ActRole: "member.access_changed",
	d.ActTarget: "target.updated", d.ActSettings: "settings.changed",
}

// entityPaths is where each record opens in the admin UI.
var entityPaths = map[string]string{"order": "/orders?view=%d", "customer": "/customers?view=%d", "product": "/products?edit=%d", "member": "/team?edit=%d"}

func newEventID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "evt_" + hex.EncodeToString(b[:])
}

// emit sends the activity entry to the webhook, if one is configured.
func (s *server) emit(ctx context.Context, a d.Activity) {
	if a.Kind == d.ActAuth || a.Kind == d.ActAlert {
		return
	}
	set := s.settings(ctx)
	if set.WebhookURL == "" {
		return
	}
	typ := eventTypes[a.Kind]
	if typ == "" {
		typ = "activity." + a.Kind
	}
	ev := webhookEvent{ID: newEventID(), Type: typ, CreatedAt: a.At, Message: a.Message}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now()
	}
	ev.Actor.ID, ev.Actor.Name = a.ActorID, a.Actor
	if a.Entity != "" && a.EntityID != 0 {
		ev.Entity = &struct {
			Type string `json:"type"`
			ID   int64  `json:"id"`
		}{a.Entity, a.EntityID}
		if p, ok := entityPaths[a.Entity]; ok && set.PublicBaseURL != "" {
			ev.URL = set.PublicBaseURL + fmt.Sprintf(p, a.EntityID)
		}
	}
	if s.hooks.inflight.Add(1) > maxInflight {
		s.hooks.inflight.Add(-1)
		s.log.Warn("webhook queue full, event dropped", "type", typ)
		return
	}
	go func() {
		defer s.hooks.inflight.Add(-1)
		s.send(ev, set)
	}()
}

// send delivers one event, retrying with backoff, and records the outcome.
func (s *server) send(ev webhookEvent, set d.Settings) delivery {
	body, _ := json.Marshal(ev)
	dl := delivery{ID: ev.ID, Type: ev.Type, URL: set.WebhookURL, At: time.Now()}
	start := time.Now()
	for attempt := 1; attempt <= webhookAttempts; attempt++ {
		dl.Attempts = attempt
		status, err := s.post(set, ev, body)
		dl.Status, dl.Error = status, ""
		if err != nil {
			dl.Error = err.Error()
		} else if status < 200 || status >= 300 {
			dl.Error = "receiver answered " + strconv.Itoa(status)
		}
		if dl.Error == "" {
			dl.OK = true
			break
		}
		if attempt < webhookAttempts {
			time.Sleep(s.hooks.backoff[min(attempt-1, len(s.hooks.backoff)-1)])
		}
	}
	dl.DurationMs = float64(time.Since(start).Microseconds()) / 1000
	s.hooks.mu.Lock()
	s.hooks.recent = append([]delivery{dl}, s.hooks.recent...)
	if len(s.hooks.recent) > deliveriesKept {
		s.hooks.recent = s.hooks.recent[:deliveriesKept]
	}
	s.hooks.mu.Unlock()
	if !dl.OK {
		s.log.Warn("webhook delivery failed", "type", ev.Type, "url", set.WebhookURL, "error", dl.Error, "attempts", dl.Attempts)
	}
	return dl
}

func (s *server) post(set d.Settings, ev webhookEvent, body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, set.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoAdmin-Webhook/1")
	req.Header.Set("X-GoAdmin-Event", ev.Type)
	req.Header.Set("X-GoAdmin-Delivery", ev.ID)
	req.Header.Set("X-GoAdmin-Timestamp", ts)
	if set.WebhooksSigned && set.WebhookSecret != "" {
		req.Header.Set("X-GoAdmin-Signature", "sha256="+sign(set.WebhookSecret, ts, body))
	}
	res, err := s.hooks.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	return res.StatusCode, nil
}

// sign is what receivers recompute: HMAC-SHA256 over "<timestamp>.<body>".
func sign(secret, timestamp string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(timestamp + "."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func newWebhookSecret() string {
	var b [24]byte
	_, _ = rand.Read(b[:])
	return "whsec_" + hex.EncodeToString(b[:])
}

// listDeliveries: GET /api/v1/settings/webhook/deliveries
func (s *server) listDeliveries(w http.ResponseWriter, _ *http.Request) {
	s.hooks.mu.Lock()
	items := append([]delivery{}, s.hooks.recent...)
	s.hooks.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// testWebhook: POST /api/v1/settings/webhook/test sends a sample event and
// returns how it went, so the URL and signature can be checked from Settings.
func (s *server) testWebhook(w http.ResponseWriter, r *http.Request) {
	set := s.settings(r.Context())
	if set.WebhookURL == "" {
		writeError(w, http.StatusConflict, "set a webhook URL first")
		return
	}
	me, _ := CurrentMember(r.Context())
	ev := webhookEvent{ID: newEventID(), Type: "webhook.test", CreatedAt: time.Now(), Message: "Test event from " + me.Name}
	ev.Actor.ID, ev.Actor.Name = me.ID, me.Name
	writeJSON(w, http.StatusOK, s.send(ev, set))
}

// rotateWebhookSecret: POST /api/v1/settings/webhook/rotate-secret
func (s *server) rotateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	set := s.settings(r.Context())
	set.WebhookSecret = newWebhookSecret()
	if err := s.store.SaveSettings(r.Context(), set); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.rememberSettings(set)
	me, _ := CurrentMember(r.Context())
	s.record(r.Context(), me, d.ActSettings, "", 0, "rotated the webhook signing secret")
	writeJSON(w, http.StatusOK, s.view(set))
}
