package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// audit records what the signed-in member just did. It runs after the change
// succeeded; a failure to log is reported but never fails the request.
func (s *server) audit(ctx context.Context, kind, entity string, entityID int64, format string, args ...any) {
	me, _ := CurrentMember(ctx)
	s.auditAs(ctx, me, kind, entity, entityID, format, args...)
}

func (s *server) auditAs(ctx context.Context, actor d.Member, kind, entity string, entityID int64, format string, args ...any) {
	_, err := s.store.RecordActivity(ctx, d.Activity{
		Kind: kind, ActorID: actor.ID, Actor: actor.Name, Message: fmt.Sprintf(format, args...),
		Entity: entity, EntityID: entityID,
	})
	if err != nil {
		s.log.WarnContext(ctx, "record activity", "kind", kind, "err", err)
	}
}

// auditProductUpdate describes an edit: a status change reads as publish or
// archive, otherwise the changed fields are listed.
func (s *server) auditProductUpdate(ctx context.Context, old, p d.Product) {
	switch {
	case old.Status != p.Status && p.Status == d.ProductActive:
		s.audit(ctx, d.ActPublish, "product", p.ID, "published %s", p.Name)
		return
	case old.Status != p.Status && p.Status == d.ProductArchived:
		s.audit(ctx, d.ActProduct, "product", p.ID, "archived %s", p.Name)
		return
	}
	var changes []string
	if old.PriceCents != p.PriceCents {
		changes = append(changes, fmt.Sprintf("price %s → %s", d.USD(old.PriceCents), d.USD(p.PriceCents)))
	}
	if old.Stock != p.Stock {
		changes = append(changes, fmt.Sprintf("stock %d → %d", old.Stock, p.Stock))
	}
	if old.Status != p.Status {
		changes = append(changes, fmt.Sprintf("status %s → %s", old.Status, p.Status))
	}
	if old.Name != p.Name {
		changes = append(changes, "renamed from "+old.Name)
	}
	if old.Description != p.Description {
		changes = append(changes, "description")
	}
	if len(changes) == 0 {
		s.audit(ctx, d.ActProduct, "product", p.ID, "updated %s", p.Name)
		return
	}
	s.audit(ctx, d.ActProduct, "product", p.ID, "updated %s: %s", p.Name, strings.Join(changes, ", "))
}

func (s *server) auditBulk(ctx context.Context, action d.BulkAction, affected []d.Product) {
	verb := map[d.BulkAction]string{d.BulkPublish: "published", d.BulkArchive: "archived", d.BulkDelete: "deleted"}[action]
	kind := d.ActProduct
	if action == d.BulkPublish {
		kind = d.ActPublish
	}
	switch len(affected) {
	case 0:
	case 1:
		p := affected[0]
		if action == d.BulkDelete {
			s.audit(ctx, kind, "", 0, "%s %s", verb, p.Name)
		} else {
			s.audit(ctx, kind, "product", p.ID, "%s %s", verb, p.Name)
		}
	default:
		s.audit(ctx, kind, "", 0, "%s %d products", verb, len(affected))
	}
}

// customerName is for log messages; it falls back to the id.
func (s *server) customerName(ctx context.Context, id int64) string {
	if det, err := s.store.GetCustomer(ctx, id); err == nil {
		return det.Customer.Name
	}
	return fmt.Sprintf("customer #%d", id)
}

// listActivity: GET /api/v1/activity?actor=&kind=&q=&limit=&offset=
func (s *server) listActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind != "" && !slices.Contains(d.ActivityKinds, kind) {
		writeError(w, http.StatusBadRequest, "unknown activity kind")
		return
	}
	items, total, err := s.store.ListActivity(r.Context(), d.ActivityFilter{
		ActorID: int64(queryInt(r, "actor", 0)),
		Kind:    kind,
		Query:   q.Get("q"),
		Limit:   min(max(queryInt(r, "limit", 50), 1), 200),
		Offset:  max(queryInt(r, "offset", 0), 0),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "kinds": d.ActivityKinds})
}
