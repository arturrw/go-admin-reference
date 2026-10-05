package seed

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// seedActivity writes a week of audit-log history that points at records in
// the dataset, so every entry opens something real.
func seedActivity(s *Dataset, now time.Time) []d.Activity {
	var out []d.Activity
	member := map[string]d.Member{}
	for _, m := range s.Members {
		member[m.Name] = m
	}
	add := func(at time.Time, kind, actor, msg, entity string, entityID int64) {
		if at.After(now) || now.Sub(at) > 7*24*time.Hour {
			return
		}
		out = append(out, d.Activity{
			Kind: kind, ActorID: member[actor].ID, Actor: actor, Message: msg,
			Entity: entity, EntityID: entityID, At: at,
		})
	}
	ago := func(dur time.Duration) time.Time { return now.Add(-dur) }
	product := func(name string) d.Product {
		for _, p := range s.Products {
			if p.Name == name {
				return p
			}
		}
		return s.Products[0]
	}
	nth := func(match func(d.Product) bool, n int) d.Product {
		for _, p := range s.Products {
			if match(p) {
				if n == 0 {
					return p
				}
				n--
			}
		}
		return s.Products[0]
	}

	// The handful the dashboard has always opened with.
	add(ago(14*time.Minute), d.ActRole, "Mark Liu", "changed Sofia Rossi's role to editor", "member", 3)
	pw := product("Pulse Watch Ultra")
	add(ago(38*time.Minute), d.ActPublish, "Yuki Tanaka", "published "+pw.Name, "product", pw.ID)
	add(ago(2*time.Hour), d.ActDeploy, "CI", "rolled out v1.4.2 to 3/3 pods", "", 0)
	low := nth(func(p d.Product) bool { return p.Stock > 0 && p.Stock < 10 }, 0)
	add(ago(3*time.Hour), d.ActStock, "Inventory", fmt.Sprintf("%s dropped below 10 units", low.Name), "product", low.ID)

	// Fulfilment and refunds follow the orders themselves.
	shippers := []string{"Mark Liu", "Omar Haddad", "Yuki Tanaka"}
	shipped := 0
	for _, o := range s.Orders {
		switch {
		case o.Status == d.OrderShipped && shipped < 8:
			add(o.PlacedAt.Add(5*time.Hour+time.Duration(o.ID%50)*time.Minute), d.ActOrder, shippers[shipped%len(shippers)],
				fmt.Sprintf("marked order #%d as shipped", o.ID), "order", o.ID)
			shipped++
		case o.Refund != nil:
			add(o.Refund.At, d.ActRefund, o.Refund.By,
				fmt.Sprintf("refunded order #%d (%s) — %s", o.ID, d.USD(o.TotalCents), o.Refund.Reason), "order", o.ID)
		}
	}

	// Support notes on customers.
	for _, c := range s.Customers {
		for _, n := range c.Notes {
			add(n.At, d.ActNote, n.Author, "added a note on "+c.Name, "customer", c.ID)
		}
	}

	// Catalogue work.
	edited := nth(func(p d.Product) bool { return p.Status == d.ProductActive }, 3)
	add(ago(26*time.Hour), d.ActProduct, "Omar Haddad", "updated the price of "+edited.Name, "product", edited.ID)
	photo := nth(func(p d.Product) bool { return p.Status == d.ProductActive }, 7)
	add(ago(31*time.Hour), d.ActImage, "Yuki Tanaka", "uploaded 2 images to "+photo.Name, "product", photo.ID)
	archived := nth(func(p d.Product) bool { return p.Status == d.ProductArchived }, 0)
	add(ago(52*time.Hour), d.ActProduct, "Mark Liu", "archived "+archived.Name, "product", archived.ID)
	draft := nth(func(p d.Product) bool { return p.Status == d.ProductDraft }, 0)
	add(ago(76*time.Hour), d.ActProduct, "Yuki Tanaka", "created "+draft.Name, "product", draft.ID)
	add(ago(5*24*time.Hour), d.ActImport, "Yuki Tanaka", "imported 12 products from CSV (4 created, 8 updated)", "", 0)

	// Team and workspace.
	add(ago(2*24*time.Hour+3*time.Hour), d.ActTeam, "Artur DCS", "invited Sofia Rossi as viewer", "member", 3)
	add(ago(4*24*time.Hour), d.ActTarget, "Artur DCS", "set the Q4 target to $1,200,000.00", "", 0)
	add(ago(6*24*time.Hour), d.ActTeam, "Mark Liu", "suspended Lena Kraft", "member", 5)
	add(ago(5*24*time.Hour+2*time.Hour), d.ActRole, "Artur DCS", "changed Diego Vega's access: granted Edit products", "member", 8)
	add(ago(6*24*time.Hour+5*time.Hour), d.ActRole, "Artur DCS", "changed Omar Haddad's access: revoked Edit settings", "member", 6)
	add(ago(3*24*time.Hour), d.ActSettings, "Mark Liu", "set the log level to debug", "", 0)
	add(ago(3*24*time.Hour-40*time.Minute), d.ActSettings, "Mark Liu", "set the log level to info", "", 0)
	add(ago(30*time.Hour), d.ActDeploy, "CI", "rolled out v1.4.1 to 3/3 pods", "", 0)

	// Sign-ins just before each member's last activity.
	for _, m := range s.Members {
		if m.LastActiveAt != nil {
			add(m.LastActiveAt.Add(-time.Duration(20+m.ID*7)*time.Minute), d.ActAuth, m.Name, "signed in", "member", m.ID)
		}
	}

	slices.SortFunc(out, func(a, b d.Activity) int { return cmp.Compare(a.At.UnixNano(), b.At.UnixNano()) })
	for i := range out {
		out[i].ID = int64(i + 1) // oldest first
	}
	return out
}
