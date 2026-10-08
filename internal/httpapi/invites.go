package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// Invitations. Inviting a member creates them as "invited" with no password
// and hands the inviter a one-time link. The server cannot send email, so the
// inviter delivers the link. Only a hash of the token is stored, and the link
// works for a week or until a newer one replaces it.

type inviteInfo struct {
	Token     string    `json:"token"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// issueInvite creates a fresh invitation link for an invited member.
func (s *server) issueInvite(ctx context.Context, r *http.Request, m d.Member) (inviteInfo, error) {
	token, err := auth.NewToken()
	if err != nil {
		return inviteInfo{}, err
	}
	expires := time.Now().Add(d.InviteLifetime)
	if err := s.store.SetInvite(ctx, m.ID, auth.HashToken(token), expires); err != nil {
		return inviteInfo{}, err
	}
	return inviteInfo{Token: token, URL: s.baseURL(ctx, r) + "/invite/" + token, ExpiresAt: expires}, nil
}

// baseURL is the public address if one is configured, else the one this
// request came in on.
func (s *server) baseURL(ctx context.Context, r *http.Request) string {
	if u := s.settings(ctx).PublicBaseURL; u != "" {
		return u
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// resendInvite: POST /api/v1/team/{id}/invite replaces the invitation link.
func (s *server) resendInvite(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	target, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	me, _ := CurrentMember(r.Context())
	switch {
	case target.Status != d.MemberInvited:
		writeError(w, http.StatusConflict, "this member has already accepted their invitation")
		return
	case target.Role == d.RoleAdmin && me.Role != d.RoleOwner:
		writeError(w, http.StatusForbidden, "only the owner can invite admins")
		return
	}
	inv, err := s.issueInvite(r.Context(), r, target)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActTeam, "member", target.ID, "issued a new invitation link for %s", target.Name)
	writeJSON(w, http.StatusOK, map[string]any{"invite": inv})
}

// inviteMember finds the invited member behind a link, or writes the 404.
func (s *server) inviteMember(w http.ResponseWriter, r *http.Request) (d.Member, bool) {
	m, err := s.store.MemberByInvite(r.Context(), auth.HashToken(r.PathValue("token")))
	if err != nil {
		writeError(w, http.StatusNotFound, "this invitation is invalid or has expired")
		return d.Member{}, false
	}
	return m, true
}

// inviteDetails: GET /api/v1/auth/invite/{token} tells the invitee what they
// were invited to, for the page they land on.
func (s *server) inviteDetails(w http.ResponseWriter, r *http.Request) {
	m, ok := s.inviteMember(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": m.Name, "email": m.Email, "role": m.Role, "serviceName": s.settings(r.Context()).ServiceName, "expiresAt": m.InviteExpiresAt,
	})
}

// acceptInvite: POST /api/v1/auth/invite/{token} {name, password} activates
// the account and signs the member in.
func (s *server) acceptInvite(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "cross-origin request rejected")
		return
	}
	var in struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	m, ok := s.inviteMember(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = m.Name
	}
	if len([]rune(name)) > 80 {
		s.writeDomainError(w, r, d.NewValidationError("name", "must be at most 80 characters"))
		return
	}
	if err := d.ValidateNewPassword(in.Password, m.Email); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if s.settings(r.Context()).Maintenance && !d.CanUseDuringMaintenance(m.Role) {
		writeMaintenance(w)
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	active, err := s.store.AcceptInvite(r.Context(), m.ID, name, hash)
	if err != nil {
		s.writeDomainError(w, r, err) // a second use of the link finds nothing: 404
		return
	}
	token, err := s.sessions.Create(r.Context(), active.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.setCookie(w, token)
	if info := reqInfoFrom(r.Context()); info != nil {
		info.actor, info.role = active.Email, string(active.Role)
	}
	s.auditAs(r.Context(), active, d.ActTeam, "member", active.ID, "accepted the invitation and joined as %s", active.Role)
	writeJSON(w, http.StatusOK, newMe(active))
}
