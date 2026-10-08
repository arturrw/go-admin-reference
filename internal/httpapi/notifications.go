package httpapi

import (
	"net/http"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// notificationCount is how many entries the bell shows (and how far "unread" counts).
const notificationCount = 30

// unreadWindow is how far back entries count as unread for a member who has
// never opened their notifications.
const unreadWindow = 24 * time.Hour

type notification struct {
	d.Activity
	Unread bool `json:"unread"`
}

// notifications: GET /api/v1/notifications is the bell. It is the activity
// feed from everyone else, plus security alerts for the signed-in member.
// Entries newer than the last time they opened it are unread.
func (s *server) notifications(w http.ResponseWriter, r *http.Request) {
	me, _ := CurrentMember(r.Context())
	if me.ID == 0 { // an API key has no inbox
		writeJSON(w, http.StatusOK, map[string]any{"items": []notification{}, "unread": 0})
		return
	}
	items, _, err := s.store.ListActivity(r.Context(), d.ActivityFilter{NotifyMember: me.ID, Limit: notificationCount})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	since := time.Now().Add(-unreadWindow)
	if me.NotificationsReadAt != nil {
		since = *me.NotificationsReadAt
	}
	out := make([]notification, len(items))
	unread := 0
	for i, a := range items {
		out[i] = notification{Activity: a, Unread: a.At.After(since)}
		if out[i].Unread {
			unread++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "unread": unread})
}

// readNotifications: POST /api/v1/notifications/read marks everything read.
func (s *server) readNotifications(w http.ResponseWriter, r *http.Request) {
	me, _ := CurrentMember(r.Context())
	if me.ID != 0 {
		if err := s.store.MarkNotificationsRead(r.Context(), me.ID, time.Now()); err != nil {
			s.writeDomainError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
