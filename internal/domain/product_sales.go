package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// A product's sales are never stored: "sold in 30 days", its revenue and the
// 14-day sparkline are derived from orders, the same Sale rows (and the same
// rule about which orders count) the dashboard uses, so they cannot disagree.
const (
	SalesDays = 30 // UTC calendar days ending today
	TrendDays = 14
)

// SalesWindowStart is the first moment orders count toward 30-day sales.
func SalesWindowStart(now time.Time) time.Time { return utcDay(now).AddDate(0, 0, -(SalesDays - 1)) }

// ProductSales is what orders say about one product.
type ProductSales struct {
	Sold         int
	RevenueCents int64
	Trend        []int // units per day, oldest first
	SoldToday    int
	RevenueToday int64
}

// SalesByProduct tallies the billable orders of the last 30 days by product.
func SalesByProduct(now time.Time, sales []Sale) map[int64]*ProductSales {
	out := map[int64]*ProductSales{}
	start, today := SalesWindowStart(now), utcDay(now)
	for _, s := range sales {
		if !s.Status.Billable() || s.PlacedAt.Before(start) {
			continue
		}
		day := int(today.Sub(utcDay(s.PlacedAt)) / (24 * time.Hour)) // 0 = today
		for _, it := range s.Items {
			if it.ProductID == 0 {
				continue
			}
			ps := out[it.ProductID]
			if ps == nil {
				ps = &ProductSales{Trend: make([]int, TrendDays)}
				out[it.ProductID] = ps
			}
			ps.Sold += it.Qty
			ps.RevenueCents += it.PriceCents * int64(it.Qty)
			if day == 0 {
				ps.SoldToday += it.Qty
				ps.RevenueToday += it.PriceCents * int64(it.Qty)
			}
			if day >= 0 && day < TrendDays {
				ps.Trend[TrendDays-1-day] += it.Qty
			}
		}
	}
	return out
}

// ApplySales fills the sales fields of products from their tally; products
// nobody bought get zeros.
func ApplySales(products []Product, by map[int64]*ProductSales) {
	for i := range products {
		p := &products[i]
		p.Sold30d, p.Revenue30dCents, p.SoldToday, p.RevenueToday, p.Trend = 0, 0, 0, 0, make([]int, TrendDays)
		if ps := by[p.ID]; ps != nil {
			p.Sold30d, p.Revenue30dCents, p.Trend = ps.Sold, ps.RevenueCents, slices.Clone(ps.Trend)
			p.SoldToday, p.RevenueToday = ps.SoldToday, ps.RevenueToday
		}
	}
}

// SortProducts orders a list the way the catalogue offers: by units sold,
// price (high first), stock (low first), name, or by default revenue.
func SortProducts(ps []Product, sort string) {
	slices.SortStableFunc(ps, func(a, b Product) int {
		var c int
		switch sort {
		case "sales":
			c = cmp.Compare(b.Sold30d, a.Sold30d)
		case "price":
			c = cmp.Compare(b.PriceCents, a.PriceCents)
		case "stock":
			c = cmp.Compare(a.Stock, b.Stock)
		case "name":
			c = strings.Compare(a.Name, b.Name)
		default:
			c = cmp.Compare(b.Revenue30dCents, a.Revenue30dCents)
		}
		return cmp.Or(c, cmp.Compare(a.ID, b.ID))
	})
}
