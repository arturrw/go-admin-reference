package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/arturrw/go-admin-reference/internal/auth"
	d "github.com/arturrw/go-admin-reference/internal/domain"
)

const memberKey ctxKey = 100

// CurrentMember returns the signed-in member set by authorize.
func CurrentMember(ctx context.Context) (d.Member, bool) {
	m, ok := ctx.Value(memberKey).(d.Member)
	return m, ok
}

// authorize resolves the session cookie to an active member, rejects
// cross-origin writes and enforces perm ("" = any signed-in member).
func (s *server) authorize(perm d.Permission, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.CookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		id, err := s.sessions.Lookup(c.Value)
		if err != nil {
			s.clearCookie(w)
			writeError(w, http.StatusUnauthorized, "session expired")
			return
		}
		m, err := s.store.GetMember(r.Context(), id)
		if err != nil || m.Status != d.MemberActive {
			s.sessions.Delete(c.Value)
			s.clearCookie(w)
			writeError(w, http.StatusUnauthorized, "account is not active")
			return
		}
		if info := reqInfoFrom(r.Context()); info != nil {
			info.actor, info.role = m.Email, string(m.Role)
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "cross-origin request rejected")
			return
		}
		if perm != "" && !m.Role.Can(perm) {
			writeError(w, http.StatusForbidden, "you don't have permission to do this")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), memberKey, m)))
	})
}

// sameOrigin is a CSRF defence on top of SameSite=Lax cookies: browsers
// always send Origin on cross-site POST/PUT/PATCH/DELETE.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser client
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

func (s *server) setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: token, Path: "/",
		MaxAge: int(s.sessions.TTL().Seconds()), HttpOnly: true,
		Secure: !s.isDev(), SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: !s.isDev(), SameSite: http.SameSiteLaxMode})
}

type meResponse struct {
	User        d.Member       `json:"user"`
	Permissions []d.Permission `json:"permissions"`
}

func newMe(m d.Member) meResponse {
	perms := d.RolePermissions[m.Role]
	if perms == nil {
		perms = []d.Permission{}
	}
	return meResponse{User: m, Permissions: perms}
}

// A valid hash of an unguessable password, checked when the email is unknown
// so both failure paths take the same time.
var dummyHash, _ = auth.HashPassword("timing-equaliser")

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin request rejected")
		return
	}
	m, err := s.store.MemberByEmail(r.Context(), strings.TrimSpace(in.Email))
	if err != nil && !errors.Is(err, d.ErrNotFound) {
		s.writeDomainError(w, r, err)
		return
	}
	hash := m.PasswordHash // empty for unknown emails and pending invites
	if hash == "" {
		hash = dummyHash
	}
	if !auth.CheckPassword(hash, in.Password) || m.PasswordHash == "" {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if m.Status != d.MemberActive {
		writeError(w, http.StatusForbidden, "this account is "+string(m.Status))
		return
	}
	token, err := s.sessions.Create(m.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.setCookie(w, token)
	s.store.TouchMember(r.Context(), m.ID)
	if info := reqInfoFrom(r.Context()); info != nil {
		info.actor, info.role = m.Email, string(m.Role)
	}
	writeJSON(w, http.StatusOK, newMe(m))
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		s.sessions.Delete(c.Value)
	}
	s.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	m, _ := CurrentMember(r.Context())
	writeJSON(w, http.StatusOK, newMe(m))
}

// demoAccounts powers the one-click sign-in on the login page. Development only.
func (s *server) demoAccounts(w http.ResponseWriter, r *http.Request) {
	if !s.isDev() || s.demoPassword == "" {
		writeError(w, http.StatusNotFound, "no such endpoint")
		return
	}
	members, err := s.store.ListMembers(r.Context(), "")
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	type account struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  d.Role `json:"role"`
	}
	out := []account{}
	seen := map[d.Role]bool{}
	for _, m := range members {
		if m.Status == d.MemberActive && m.PasswordHash != "" && !seen[m.Role] {
			seen[m.Role] = true
			out = append(out, account{m.Name, m.Email, m.Role})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"password": s.demoPassword, "accounts": out})
}

func (s *server) roles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"roles":       d.Roles,
		"permissions": d.Permissions,
		"matrix":      d.RolePermissions,
	})
}
