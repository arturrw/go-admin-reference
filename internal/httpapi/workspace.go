package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

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
	return v
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
	writeJSON(w, http.StatusOK, settingsView{Settings: s.settings(r.Context()), ListenAddr: s.addr, Env: s.env})
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
	for _, c := range before.Changes(after) {
		s.audit(r.Context(), d.ActSettings, "", 0, "%s", c)
	}
	writeJSON(w, http.StatusOK, settingsView{Settings: after, ListenAddr: s.addr, Env: s.env})
}
