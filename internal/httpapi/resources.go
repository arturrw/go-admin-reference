package httpapi

import (
	"net/http"
	"slices"

	"github.com/arturrw/go-admin-reference/internal/domain"
)

// ── Products ────────────────────────────────────────────────────────────────

func (s *server) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListProducts(r.Context(), domain.ProductFilter{
		Query:    q.Get("q"),
		Category: domain.Category(q.Get("category")),
		Status:   domain.ProductStatus(q.Get("status")),
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

func (s *server) createProduct(w http.ResponseWriter, r *http.Request) {
	var in domain.ProductInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
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
	var in domain.ProductInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
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
	if err := s.store.DeleteProduct(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) bulkProducts(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs    []int64           `json:"ids"`
		Action domain.BulkAction `json:"action"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.IDs) == 0 || len(in.IDs) > 500 {
		writeError(w, http.StatusBadRequest, "ids must contain 1–500 product ids")
		return
	}
	if !slices.Contains([]domain.BulkAction{domain.BulkPublish, domain.BulkArchive, domain.BulkDelete}, in.Action) {
		writeError(w, http.StatusBadRequest, "action must be publish, archive or delete")
		return
	}
	n, err := s.store.BulkProducts(r.Context(), in.IDs, in.Action)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"affected": n})
}

// ── Orders ──────────────────────────────────────────────────────────────────

func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListOrders(r.Context(), domain.OrderFilter{
		Query:  q.Get("q"),
		Status: domain.OrderStatus(q.Get("status")),
		Limit:  queryInt(r, "limit", 0),
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
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": counts})
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
		Status domain.OrderStatus `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !slices.Contains(domain.OrderStatuses, in.Status) {
		s.writeDomainError(w, r, &domain.ValidationError{Fields: map[string]string{"status": "is not a valid order status"}})
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
	items, err := s.store.ListCustomers(r.Context(), domain.CustomerFilter{Query: q.Get("q"), Segment: q.Get("segment")})
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

// ── Team ────────────────────────────────────────────────────────────────────

func (s *server) listMembers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListMembers(r.Context(), domain.Role(r.URL.Query().Get("role")))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) createMember(w http.ResponseWriter, r *http.Request) {
	var in domain.MemberInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
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
	var in domain.MemberInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		s.writeDomainError(w, r, err)
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
	if err := s.store.DeleteMember(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
