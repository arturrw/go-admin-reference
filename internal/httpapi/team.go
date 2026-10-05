package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

type memberDetail struct {
	Member      d.Member       `json:"member"`
	Permissions []d.Permission `json:"permissions"` // effective
	Online      bool           `json:"online"`
}

func newMemberDetail(m d.Member) memberDetail {
	online := m.Status == d.MemberActive && m.LastActiveAt != nil && time.Since(*m.LastActiveAt) < d.OnlineWithin
	return memberDetail{Member: m, Permissions: m.Permissions(), Online: online}
}

// getMember: GET /api/v1/team/{id} — profile, effective permissions and presence.
func (s *server) getMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	m, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newMemberDetail(m))
}

// setMemberAccess: PUT /api/v1/team/{id}/access — per-member exceptions to the
// role. Only the owner manages them, and the owner's own access is fixed.
func (s *server) setMemberAccess(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in d.MemberAccess
	if !decodeJSON(w, r, &in) {
		return
	}
	if me, _ := CurrentMember(r.Context()); me.Role != d.RoleOwner {
		writeError(w, http.StatusForbidden, "only the owner can change individual permissions")
		return
	}
	target, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if target.Role == d.RoleOwner {
		writeError(w, http.StatusForbidden, "the owner always has full access")
		return
	}
	access, err := in.Normalize(target.Role)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	m, err := s.store.SetMemberAccess(r.Context(), id, access)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if change := accessChange(target, m); change != "" {
		s.audit(r.Context(), d.ActRole, "member", m.ID, "changed %s's access: %s", m.Name, change)
	}
	writeJSON(w, http.StatusOK, newMemberDetail(m))
}

// accessChange describes what changed between two sets of exceptions.
func accessChange(before, after d.Member) string {
	var granted, revoked, restored []string
	was, now := before.Permissions(), after.Permissions()
	for _, p := range d.Permissions {
		had, has := slices.Contains(was, p.Key), slices.Contains(now, p.Key)
		switch {
		case !had && has && !after.Role.Can(p.Key):
			granted = append(granted, p.Label)
		case had && !has:
			revoked = append(revoked, p.Label)
		case had != has:
			restored = append(restored, p.Label)
		}
	}
	var parts []string
	if len(granted) > 0 {
		parts = append(parts, "granted "+strings.Join(granted, ", "))
	}
	if len(revoked) > 0 {
		parts = append(parts, "revoked "+strings.Join(revoked, ", "))
	}
	if len(restored) > 0 {
		parts = append(parts, "restored "+strings.Join(restored, ", "))
	}
	return strings.Join(parts, "; ")
}

// setMemberStatus: PUT /api/v1/team/{id}/status — suspend or reactivate.
// Suspending signs the member out everywhere.
func (s *server) setMemberStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Status d.MemberStatus `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Status != d.MemberActive && in.Status != d.MemberSuspended {
		s.writeDomainError(w, r, d.NewValidationError("status", "must be active or suspended"))
		return
	}
	me, _ := CurrentMember(r.Context())
	if me.ID == id {
		writeError(w, http.StatusForbidden, "you cannot change your own status")
		return
	}
	target, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	switch {
	case target.Role == d.RoleOwner:
		writeError(w, http.StatusForbidden, "the owner cannot be suspended")
		return
	case target.Role == d.RoleAdmin && me.Role != d.RoleOwner:
		writeError(w, http.StatusForbidden, "only the owner can suspend admins")
		return
	case target.Status == d.MemberInvited:
		writeError(w, http.StatusConflict, "the invite has not been accepted yet")
		return
	case target.Status == in.Status:
		writeJSON(w, http.StatusOK, newMemberDetail(target))
		return
	}
	m, err := s.store.SetMemberStatus(r.Context(), id, in.Status)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if m.Status == d.MemberSuspended {
		s.sessions.DeleteMember(r.Context(), id)
		s.audit(r.Context(), d.ActTeam, "member", m.ID, "suspended %s", m.Name)
	} else {
		s.audit(r.Context(), d.ActTeam, "member", m.ID, "reactivated %s", m.Name)
	}
	writeJSON(w, http.StatusOK, newMemberDetail(m))
}
