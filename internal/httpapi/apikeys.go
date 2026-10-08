package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// API keys let other servers call /api/v1 with `Authorization: Bearer <key>`.
// A key acts as a viewer (read) or an editor (write), never as an admin, so it
// can't reach settings and a leaked key can't create more keys.

func (s *server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.ListAPIKeys(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	type item struct {
		d.APIKey
		Masked string `json:"masked"`
	}
	out := make([]item, len(keys))
	for i, k := range keys {
		out[i] = item{k, k.Masked()}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// createAPIKey returns the secret once, in the response; only its hash is kept.
func (s *server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var in d.APIKeyInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	secret := d.KeyPrefix + hex.EncodeToString(raw[:])
	me, _ := CurrentMember(r.Context())
	k, err := s.store.CreateAPIKey(r.Context(), d.APIKey{
		Name: in.Name, Scope: in.Scope, Last4: secret[len(secret)-4:], CreatedBy: me.Name, Hash: auth.HashToken(secret),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActSettings, "", 0, "created API key “%s” (%s access)", k.Name, k.Scope)
	writeJSON(w, http.StatusCreated, map[string]any{"key": k, "masked": k.Masked(), "secret": secret})
}

func (s *server) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	k, err := s.store.RevokeAPIKey(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActSettings, "", 0, "revoked API key “%s”", k.Name)
	w.WriteHeader(http.StatusNoContent)
}

// bearerKey extracts a key from `Authorization: Bearer ga_live_…`.
func bearerKey(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if len(h) < 8 || !strings.EqualFold(h[:7], "bearer ") {
		return "", false
	}
	return strings.TrimSpace(h[7:]), true
}

// keyPrincipal resolves a key to the member it acts as, noting that it was
// used (at most once a minute, like member presence).
func (s *server) keyPrincipal(ctx context.Context, secret string) (d.Member, bool) {
	k, err := s.store.APIKeyByHash(ctx, auth.HashToken(secret))
	if err != nil {
		return d.Member{}, false
	}
	if k.LastUsedAt == nil || time.Since(*k.LastUsedAt) > time.Minute {
		s.store.TouchAPIKey(ctx, k.ID)
	}
	return k.Principal(), true
}
