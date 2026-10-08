package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// settingsCache keeps the workspace settings in memory, refreshed from the
// store every few seconds so every request can consult them cheaply and
// several instances converge quickly.
type settingsCache struct {
	mu sync.Mutex
	v  d.Settings
	at time.Time
}

const settingsTTL = 5 * time.Second

func (s *server) settings(ctx context.Context) d.Settings {
	s.cfg.mu.Lock()
	defer s.cfg.mu.Unlock()
	if !s.cfg.at.IsZero() && time.Since(s.cfg.at) < settingsTTL {
		return s.cfg.v
	}
	v, err := s.store.GetSettings(ctx)
	if err != nil {
		s.log.WarnContext(ctx, "load settings", "err", err)
		if s.cfg.at.IsZero() {
			return d.DefaultSettings()
		}
		return s.cfg.v
	}
	s.cfg.v, s.cfg.at = v, time.Now()
	s.applySessionTTL(v)
	return v
}

// applySessionTTL makes the chosen session lifetime take effect (also on
// other instances, as they refresh the settings).
func (s *server) applySessionTTL(v d.Settings) {
	if s.sessions == nil || v.SessionTTLSeconds <= 0 {
		return
	}
	if want := time.Duration(v.SessionTTLSeconds) * time.Second; s.sessions.TTL() != want {
		s.sessions.SetTTL(want)
	}
}

// view adds what the UI shows but isn't a setting: the effective session
// lifetime (SESSION_TTL until an admin picks one), address and environment.
func (s *server) view(v d.Settings) settingsView {
	if s.sessions != nil {
		v.SessionTTLSeconds = int(s.sessions.TTL() / time.Second)
	}
	return settingsView{Settings: v, ListenAddr: s.addr, Env: s.env}
}

func (s *server) rememberSettings(v d.Settings) {
	s.cfg.mu.Lock()
	s.cfg.v, s.cfg.at = v, time.Now()
	s.cfg.mu.Unlock()
}

type settingsView struct {
	d.Settings
	ListenAddr string `json:"listenAddr"` // set by ADDR; changing it needs a restart
	Env        string `json:"env"`
}

// getSettings: GET /api/v1/settings
func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.view(s.settings(r.Context())))
}

// patchSettings: PATCH /api/v1/settings changes only the fields present.
func (s *server) patchSettings(w http.ResponseWriter, r *http.Request) {
	var p d.SettingsPatch
	if !decodeJSON(w, r, &p) {
		return
	}
	p.Normalize()
	if err := p.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	before := s.settings(r.Context())
	after := before.Apply(p)
	if err := s.store.SaveSettings(r.Context(), after); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.rememberSettings(after)
	s.applySessionTTL(after)
	me, _ := CurrentMember(r.Context())
	for _, c := range before.Changes(after) {
		s.record(r.Context(), me, d.ActSettings, "", 0, "%s", c)
	}
	writeJSON(w, http.StatusOK, s.view(after))
}

// clearRequestLog: POST /api/v1/danger/clear-request-log
func (s *server) clearRequestLog(w http.ResponseWriter, r *http.Request) {
	n := s.requests.Clear()
	s.audit(r.Context(), d.ActSettings, "", 0, "cleared the request log (%d entries)", n)
	writeJSON(w, http.StatusOK, map[string]int{"cleared": n})
}

// signOutEveryone: POST /api/v1/danger/sign-out-everyone ends every session
// but the caller's own. API keys keep working; revoke them separately.
func (s *server) signOutEveryone(w http.ResponseWriter, r *http.Request) {
	keep := ""
	if c, err := r.Cookie(auth.CookieName); err == nil {
		keep = c.Value
	}
	n := s.sessions.DeleteOthers(r.Context(), keep)
	s.audit(r.Context(), d.ActSettings, "", 0, "signed everyone else out (%d sessions)", n)
	writeJSON(w, http.StatusOK, map[string]int64{"signedOut": n})
}
