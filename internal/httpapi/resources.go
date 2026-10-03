package httpapi

import (
	"errors"
	"net/http"
	"path"
	"slices"
	"strings"

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
	p, err := s.store.UpdateProduct(r.Context(), id, in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
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
	items, total, err := s.store.ListOrders(r.Context(), d.OrderFilter{
		Query:      q.Get("q"),
		Status:     d.OrderStatus(q.Get("status")),
		CustomerID: int64(queryInt(r, "customer", 0)),
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
	writeJSON(w, http.StatusOK, o)
}

func (s *server) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Status d.OrderStatus `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !slices.Contains(d.OrderStatuses, in.Status) {
		s.writeDomainError(w, r, d.NewValidationError("status", "is not a valid order status"))
		return
	}
	o, err := s.store.UpdateOrderStatus(r.Context(), id, in.Status)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
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
	writeJSON(w, http.StatusCreated, note)
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
	writeJSON(w, http.StatusCreated, m)
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
	w.WriteHeader(http.StatusNoContent)
}
