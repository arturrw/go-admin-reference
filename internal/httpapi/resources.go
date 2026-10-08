package httpapi

import (
	"errors"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/media"
)

// ── Products ────────────────────────────────────────────────────────────────

func (s *server) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListProducts(r.Context(), d.ProductFilter{
		Query:    q.Get("q"),
		Category: d.Category(q.Get("category")),
		Status:   d.ProductStatus(q.Get("status")),
		Sort:     q.Get("sort"),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	stats, err := s.store.ProductStats(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "stats": stats})
}

func (s *server) getProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := s.store.GetProduct(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func decodeProductInput(w http.ResponseWriter, r *http.Request) (d.ProductInput, bool) {
	var in d.ProductInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Normalize()
	return in, true
}

func (s *server) createProduct(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeProductInput(w, r)
	if !ok {
		return
	}
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	p, err := s.store.CreateProduct(r.Context(), in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActProduct, "product", p.ID, "created %s", p.Name)
	writeJSON(w, http.StatusCreated, p)
}

func (s *server) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := decodeProductInput(w, r)
	if !ok {
		return
	}
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	old, err := s.store.GetProduct(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	p, err := s.store.UpdateProduct(r.Context(), id, in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auditProductUpdate(r.Context(), old, p)
	writeJSON(w, http.StatusOK, p)
}

func (s *server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := s.store.DeleteProduct(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.removeFiles(r, p.Images...)
	s.audit(r.Context(), d.ActProduct, "", 0, "deleted %s (%s)", p.Name, p.SKU)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) bulkProducts(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs    []int64      `json:"ids"`
		Action d.BulkAction `json:"action"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.IDs) == 0 || len(in.IDs) > 500 {
		writeError(w, http.StatusBadRequest, "ids must contain 1–500 product ids")
		return
	}
	if !slices.Contains([]d.BulkAction{d.BulkPublish, d.BulkArchive, d.BulkDelete}, in.Action) {
		writeError(w, http.StatusBadRequest, "action must be publish, archive or delete")
		return
	}
	affected, err := s.store.BulkProducts(r.Context(), in.IDs, in.Action)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if in.Action == d.BulkDelete {
		for _, p := range affected {
			s.removeFiles(r, p.Images...)
		}
	}
	s.auditBulk(r.Context(), in.Action, affected)
	writeJSON(w, http.StatusOK, map[string]int{"affected": len(affected)})
}

// uploadProductImage accepts multipart/form-data with a "file" field and an
// optional "alt" text.
func (s *server) uploadProductImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetProduct(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+64<<10)
	file, hdr, err := r.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "image must be at most 5 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "expected a multipart form with a \"file\" field")
		return
	}
	defer file.Close()

	name, size, err := s.media.Save(file)
	switch {
	case errors.Is(err, media.ErrUnsupportedType):
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	alt := strings.TrimSpace(r.FormValue("alt"))
	if alt == "" {
		alt = strings.TrimSuffix(hdr.Filename, path.Ext(hdr.Filename))
	}
	img := d.ProductImage{
		ID:        "up-" + strings.TrimSuffix(name, path.Ext(name)),
		URL:       "/media/uploads/" + name,
		Alt:       alt[:min(len(alt), 120)],
		SizeBytes: size,
	}
	p, err := s.store.AddProductImage(r.Context(), id, img)
	if err != nil {
		_ = s.media.Delete(name)
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActImage, "product", p.ID, "uploaded an image to %s", p.Name)
	writeJSON(w, http.StatusCreated, p)
}

func (s *server) deleteProductImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, img, err := s.store.DeleteProductImage(r.Context(), id, r.PathValue("imageId"))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.removeFiles(r, img)
	s.audit(r.Context(), d.ActImage, "product", p.ID, "removed an image from %s", p.Name)
	writeJSON(w, http.StatusOK, p)
}

func (s *server) setPrimaryImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := s.store.SetPrimaryImage(r.Context(), id, r.PathValue("imageId"))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActImage, "product", p.ID, "changed the cover image of %s", p.Name)
	writeJSON(w, http.StatusOK, p)
}

// removeFiles deletes uploaded files backing images; generated art has none.
func (s *server) removeFiles(r *http.Request, imgs ...d.ProductImage) {
	for _, img := range imgs {
		if img.Generated || s.media == nil {
			continue
		}
		if err := s.media.Delete(path.Base(img.URL)); err != nil {
			s.log.WarnContext(r.Context(), "delete image file", "url", img.URL, "err", err)
		}
	}
}

// ── Orders ──────────────────────────────────────────────────────────────────

func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := min(max(queryInt(r, "limit", 25), 1), 100)
	from, ok := queryTime(w, r, "from")
	if !ok {
		return
	}
	to, ok := queryTime(w, r, "to")
	if !ok {
		return
	}
	items, total, err := s.store.ListOrders(r.Context(), d.OrderFilter{
		Query:      q.Get("q"),
		Status:     d.OrderStatus(q.Get("status")),
		CustomerID: int64(queryInt(r, "customer", 0)),
		From:       from,
		To:         to,
		Limit:      limit,
		Offset:     max(queryInt(r, "offset", 0), 0),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	counts, err := s.store.OrderCounts(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": counts, "total": total})
}

func (s *server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	o, err := s.store.GetOrder(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	o.FillTotals()
	writeJSON(w, http.StatusOK, o)
}

func (s *server) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Status d.OrderStatus `json:"status"`
		Reason string        `json:"reason"` // required when refunding
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !slices.Contains(d.OrderStatuses, in.Status) {
		s.writeDomainError(w, r, d.NewValidationError("status", "is not a valid order status"))
		return
	}
	cur, err := s.store.GetOrder(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	me, _ := CurrentMember(r.Context())
	var refund *d.OrderRefund
	if in.Status == d.OrderRefunded {
		reason, err := d.ValidateRefundReason(in.Reason)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		if !cur.Status.Billable() {
			writeError(w, http.StatusConflict, "only paid orders can be refunded")
			return
		}
		refund = &d.OrderRefund{Reason: reason, By: me.Name, At: time.Now()}
	}
	o, err := s.store.UpdateOrderStatus(r.Context(), id, in.Status, refund)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if refund != nil {
		s.audit(r.Context(), d.ActRefund, "order", o.ID, "refunded order #%d (%s) — %s", o.ID, d.USD(o.TotalCents), refund.Reason)
	} else {
		s.audit(r.Context(), d.ActOrder, "order", o.ID, "marked order #%d as %s", o.ID, o.Status)
	}
	writeJSON(w, http.StatusOK, o)
}

// ── Customers ───────────────────────────────────────────────────────────────

func (s *server) listCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListCustomers(r.Context(), d.CustomerFilter{Query: q.Get("q"), Segment: q.Get("segment")})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	segments, err := s.store.CustomerSegments(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "segments": segments})
}

func (s *server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var in d.CustomerInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	c, err := s.store.CreateCustomer(r.Context(), in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActCustomer, "customer", c.ID, "added customer %s", c.Name)
	writeJSON(w, http.StatusCreated, c)
}

func (s *server) getCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	det, err := s.store.GetCustomer(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, det)
}

func (s *server) addCustomerNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	text, err := d.ValidateNote(in.Text)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	me, _ := CurrentMember(r.Context())
	note, err := s.store.AddCustomerNote(r.Context(), id, me.Name, text)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActNote, "customer", id, "added a note on %s", s.customerName(r.Context(), id))
	writeJSON(w, http.StatusCreated, note)
}

func (s *server) deleteCustomerNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	noteID, err := strconv.ParseInt(r.PathValue("noteId"), 10, 64)
	if err != nil || noteID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid note id")
		return
	}
	note, err := s.store.DeleteCustomerNote(r.Context(), id, noteID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.audit(r.Context(), d.ActNote, "customer", id, "deleted %s's note on %s: “%s”", note.Author, s.customerName(r.Context(), id), excerpt(note.Text, 60))
	w.WriteHeader(http.StatusNoContent)
}

// excerpt shortens text for log messages, on a rune boundary.
func excerpt(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return strings.TrimSpace(string(r[:n])) + "…"
	}
	return s
}

// ── Team ────────────────────────────────────────────────────────────────────

func (s *server) listMembers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListMembers(r.Context(), d.Role(r.URL.Query().Get("role")))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func decodeMemberInput(w http.ResponseWriter, r *http.Request) (d.MemberInput, bool) {
	var in d.MemberInput
	if !decodeJSON(w, r, &in) {
		return in, false
	}
	in.Normalize()
	return in, true
}

// Only the owner may grant the admin role.
func canAssign(me d.Member, role d.Role) bool {
	return role != d.RoleAdmin || me.Role == d.RoleOwner
}

func (s *server) createMember(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeMemberInput(w, r)
	if !ok {
		return
	}
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if me, _ := CurrentMember(r.Context()); !canAssign(me, in.Role) {
		writeError(w, http.StatusForbidden, "only the owner can grant the admin role")
		return
	}
	m, err := s.store.CreateMember(r.Context(), in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	inv, err := s.issueInvite(r.Context(), r, m)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	m.InviteExpiresAt = &inv.ExpiresAt
	s.audit(r.Context(), d.ActTeam, "member", m.ID, "invited %s as %s", m.Name, m.Role)
	writeJSON(w, http.StatusCreated, struct {
		d.Member
		Invite inviteInfo `json:"invite"`
	}{m, inv})
}

func (s *server) updateMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := decodeMemberInput(w, r)
	if !ok {
		return
	}
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	me, _ := CurrentMember(r.Context())
	target, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if me.Role != d.RoleOwner && (target.Role == d.RoleAdmin || !canAssign(me, in.Role)) {
		writeError(w, http.StatusForbidden, "only the owner can change admins")
		return
	}
	m, err := s.store.UpdateMember(r.Context(), id, in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if m.Role != target.Role {
		s.audit(r.Context(), d.ActRole, "member", m.ID, "changed %s's role from %s to %s", m.Name, target.Role, m.Role)
	} else {
		s.audit(r.Context(), d.ActTeam, "member", m.ID, "updated %s's profile", m.Name)
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *server) deleteMember(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	me, _ := CurrentMember(r.Context())
	if me.ID == id {
		writeError(w, http.StatusForbidden, "you cannot remove yourself")
		return
	}
	target, err := s.store.GetMember(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if target.Role == d.RoleAdmin && me.Role != d.RoleOwner {
		writeError(w, http.StatusForbidden, "only the owner can remove admins")
		return
	}
	if err := s.store.DeleteMember(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.sessions.DeleteMember(r.Context(), id)
	s.audit(r.Context(), d.ActTeam, "", 0, "removed %s (%s) from the team", target.Name, target.Role)
	w.WriteHeader(http.StatusNoContent)
}
