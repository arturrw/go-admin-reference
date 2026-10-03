// Package memory is an in-memory implementation of the admin store, seeded
// with deterministic fake data. It is the stand-in until the Postgres store
// (pgx + sqlc) lands; both satisfy httpapi.Store.
package memory

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

type Store struct {
	mu            sync.RWMutex
	products      []d.Product
	orders        []d.Order // newest first
	customers     []d.Customer
	members       []d.Member
	revenue       []d.RevenuePoint
	heatmap       [7][24]int
	activity      []d.Activity
	nextProductID int64
	nextMemberID  int64
	nextNoteID    int64
	now           func() time.Time
}

func New(now time.Time) *Store {
	s := &Store{now: time.Now}
	seed(s, now)
	return s
}

func contains(haystack, needle string) bool {
	return needle == "" || strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// ── Products ────────────────────────────────────────────────────────────────

func (s *Store) ListProducts(_ context.Context, f d.ProductFilter) ([]d.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]d.Product, 0, len(s.products))
	for _, p := range s.products {
		if (f.Category == "" || p.Category == f.Category) &&
			(f.Status == "" || p.Status == f.Status) &&
			contains(p.Name+" "+p.SKU+" "+p.Vendor+" "+strings.Join(p.Tags, " "), f.Query) {
			out = append(out, cloneProduct(p))
		}
	}
	slices.SortStableFunc(out, func(a, b d.Product) int {
		switch f.Sort {
		case "sales":
			return cmp.Compare(b.Sold30d, a.Sold30d)
		case "price":
			return cmp.Compare(b.PriceCents, a.PriceCents)
		case "stock":
			return cmp.Compare(a.Stock, b.Stock)
		case "name":
			return strings.Compare(a.Name, b.Name)
		default:
			return cmp.Compare(b.Revenue30dCents(), a.Revenue30dCents())
		}
	})
	return out, nil
}

func (s *Store) ProductStats(_ context.Context) (d.ProductStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := d.ProductStats{Total: len(s.products), ByCategory: map[d.Category]int{}}
	for _, p := range s.products {
		st.ByCategory[p.Category]++
		st.InventoryValueCents += int64(p.Stock) * p.PriceCents
		switch {
		case p.Stock == 0:
			st.OutOfStock++
		case p.Stock < d.LowStockThreshold:
			st.LowStock++
		}
		if p.Status == d.ProductActive {
			st.Active++
		}
	}
	return st, nil
}

func (s *Store) GetProduct(_ context.Context, id int64) (d.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := s.productIndex(id)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	return cloneProduct(s.products[i]), nil
}

func (s *Store) CreateProduct(_ context.Context, in d.ProductInput) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.skuTaken(in.SKU, 0) {
		return d.Product{}, d.NewValidationError("sku", "is already used by another product")
	}
	now := s.now()
	p := d.Product{
		ID: s.nextProductID, Hue: categoryHue(in.Category), Trend: make([]int, 14),
		Images: []d.ProductImage{}, CreatedAt: now, UpdatedAt: now,
	}
	applyProductInput(&p, in)
	s.nextProductID++
	s.products = append([]d.Product{p}, s.products...)
	return cloneProduct(p), nil
}

func (s *Store) UpdateProduct(_ context.Context, id int64, in d.ProductInput) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(id)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	if s.skuTaken(in.SKU, id) {
		return d.Product{}, d.NewValidationError("sku", "is already used by another product")
	}
	p := &s.products[i]
	if p.Category != in.Category {
		p.Hue = categoryHue(in.Category)
	}
	applyProductInput(p, in)
	p.UpdatedAt = s.now()
	return cloneProduct(*p), nil
}

// DeleteProduct removes the product and returns it so the caller can clean
// up uploaded files.
func (s *Store) DeleteProduct(_ context.Context, id int64) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(id)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	p := s.products[i]
	s.products = slices.Delete(s.products, i, i+1)
	return p, nil
}

func (s *Store) BulkProducts(_ context.Context, ids []int64, action d.BulkAction) ([]d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var affected []d.Product
	if action == d.BulkDelete {
		s.products = slices.DeleteFunc(s.products, func(p d.Product) bool {
			hit := slices.Contains(ids, p.ID)
			if hit {
				affected = append(affected, p)
			}
			return hit
		})
		return affected, nil
	}
	status := d.ProductActive
	if action == d.BulkArchive {
		status = d.ProductArchived
	}
	for i := range s.products {
		if slices.Contains(ids, s.products[i].ID) {
			s.products[i].Status = status
			s.products[i].UpdatedAt = s.now()
			affected = append(affected, s.products[i])
		}
	}
	return affected, nil
}

func (s *Store) AddProductImage(_ context.Context, productID int64, img d.ProductImage) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(productID)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	p := &s.products[i]
	if len(p.Images) >= d.MaxProductImages {
		return d.Product{}, d.NewValidationError("file", "a product can have at most 8 images")
	}
	p.Images = append(p.Images, img)
	p.UpdatedAt = s.now()
	return cloneProduct(*p), nil
}

// DeleteProductImage removes an image and returns it for file cleanup.
func (s *Store) DeleteProductImage(_ context.Context, productID int64, imageID string) (d.Product, d.ProductImage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(productID)
	if i < 0 {
		return d.Product{}, d.ProductImage{}, d.ErrNotFound
	}
	p := &s.products[i]
	j := slices.IndexFunc(p.Images, func(im d.ProductImage) bool { return im.ID == imageID })
	if j < 0 {
		return d.Product{}, d.ProductImage{}, d.ErrNotFound
	}
	img := p.Images[j]
	p.Images = slices.Delete(p.Images, j, j+1)
	p.UpdatedAt = s.now()
	return cloneProduct(*p), img, nil
}

// SetPrimaryImage moves an image to the front of the gallery.
func (s *Store) SetPrimaryImage(_ context.Context, productID int64, imageID string) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(productID)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	p := &s.products[i]
	j := slices.IndexFunc(p.Images, func(im d.ProductImage) bool { return im.ID == imageID })
	if j < 0 {
		return d.Product{}, d.ErrNotFound
	}
	img := p.Images[j]
	p.Images = slices.Insert(slices.Delete(p.Images, j, j+1), 0, img)
	p.UpdatedAt = s.now()
	return cloneProduct(*p), nil
}

func (s *Store) productIndex(id int64) int {
	return slices.IndexFunc(s.products, func(p d.Product) bool { return p.ID == id })
}

func (s *Store) skuTaken(sku string, except int64) bool {
	return slices.ContainsFunc(s.products, func(p d.Product) bool { return p.SKU == sku && p.ID != except })
}

func applyProductInput(p *d.Product, in d.ProductInput) {
	p.Name, p.SKU, p.Category, p.Vendor = in.Name, in.SKU, in.Category, in.Vendor
	p.Tags = slices.Clone(in.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	p.PriceCents, p.CompareAtCents, p.CostCents = in.PriceCents, in.CompareAtCents, in.CostCents
	p.Stock, p.WeightGrams, p.Status, p.Description = in.Stock, in.WeightGrams, in.Status, in.Description
}

// cloneProduct copies slice fields so callers cannot mutate store state.
func cloneProduct(p d.Product) d.Product {
	p.Tags = slices.Clone(p.Tags)
	p.Trend = slices.Clone(p.Trend)
	p.Images = slices.Clone(p.Images)
	return p
}

func categoryHue(c d.Category) int { return catalog[c].hue }

// ── Orders ──────────────────────────────────────────────────────────────────

func (s *Store) ListOrders(_ context.Context, f d.OrderFilter) ([]d.Order, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matched []d.Order
	for _, o := range s.orders {
		if (f.Status == "" || o.Status == f.Status) &&
			(f.CustomerID == 0 || o.Customer.ID == f.CustomerID) &&
			(contains(o.Customer.Name+" "+o.Customer.Email, f.Query) || contains("#"+strconv.FormatInt(o.ID, 10), f.Query)) {
			matched = append(matched, o)
		}
	}
	total := len(matched)
	lo := min(max(f.Offset, 0), total)
	hi := total
	if f.Limit > 0 {
		hi = min(lo+f.Limit, total)
	}
	return slices.Clone(matched[lo:hi]), total, nil
}

func (s *Store) OrderCounts(_ context.Context) (map[d.OrderStatus]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := map[d.OrderStatus]int{}
	for _, o := range s.orders {
		counts[o.Status]++
	}
	return counts, nil
}

func (s *Store) GetOrder(_ context.Context, id int64) (d.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return d.Order{}, d.ErrNotFound
}

func (s *Store) UpdateOrderStatus(_ context.Context, id int64, status d.OrderStatus) (d.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID == id {
			s.orders[i].Status = status
			if ci := s.customerIndex(s.orders[i].Customer.ID); ci >= 0 {
				s.recomputeCustomer(ci, s.now())
			}
			return s.orders[i], nil
		}
	}
	return d.Order{}, d.ErrNotFound
}

// ── Customers ───────────────────────────────────────────────────────────────

func (s *Store) ListCustomers(_ context.Context, f d.CustomerFilter) ([]d.Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []d.Customer{}
	for _, c := range s.customers {
		if (f.Segment == "" || c.Segment == f.Segment) && contains(c.Name+" "+c.Email+" "+c.Address.City, f.Query) {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b d.Customer) int { return cmp.Compare(b.LTVCents, a.LTVCents) })
	return out, nil
}

func (s *Store) CustomerSegments(_ context.Context) (map[string]d.SegmentSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]d.SegmentSummary{}
	for _, c := range s.customers {
		sum := out[c.Segment]
		sum.Count++
		sum.LTVCents += c.LTVCents
		out[c.Segment] = sum
	}
	return out, nil
}

func (s *Store) GetCustomer(_ context.Context, id int64) (d.CustomerDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ci := s.customerIndex(id)
	if ci < 0 {
		return d.CustomerDetail{}, d.ErrNotFound
	}
	c := s.customers[ci]
	c.Notes = slices.Clone(c.Notes)
	slices.SortFunc(c.Notes, func(a, b d.CustomerNote) int { return b.At.Compare(a.At) })

	det := d.CustomerDetail{Customer: c, Orders: []d.Order{}}
	products := map[int64]*d.PurchasedProduct{}
	categories := map[d.Category]int64{}
	monthly := map[string]int64{}
	for _, o := range s.orders {
		if o.Customer.ID != id {
			continue
		}
		det.Orders = append(det.Orders, o)
		if o.Status == d.OrderRefunded {
			det.Stats.Refunds++
		}
		if !o.Status.Billable() {
			continue
		}
		det.Stats.TotalSpentCents += o.TotalCents
		det.Stats.Orders++
		monthly[o.PlacedAt.Format("2006-01")] += o.TotalCents
		for _, it := range o.Items {
			det.Stats.ItemsBought += it.Qty
			categories[it.Category] += it.PriceCents * int64(it.Qty)
			pp, ok := products[it.ProductID]
			if !ok {
				pp = &d.PurchasedProduct{ProductID: it.ProductID, Name: it.Name, Category: it.Category, Hue: it.Hue, ImageURL: it.ImageURL}
				products[it.ProductID] = pp
			}
			pp.Qty += it.Qty
			pp.SpentCents += it.PriceCents * int64(it.Qty)
		}
	}
	if n := len(det.Orders); n > 0 {
		det.Stats.LastOrderAt = det.Orders[0].PlacedAt
		det.Stats.FirstOrderAt = det.Orders[n-1].PlacedAt
	}
	if det.Stats.Orders > 0 {
		det.Stats.AvgOrderCents = det.Stats.TotalSpentCents / int64(det.Stats.Orders)
	}

	det.Products = make([]d.PurchasedProduct, 0, len(products))
	for _, p := range products {
		det.Products = append(det.Products, *p)
	}
	slices.SortFunc(det.Products, func(a, b d.PurchasedProduct) int { return cmp.Compare(b.SpentCents, a.SpentCents) })

	det.Categories = []d.CategoryShare{}
	for _, cat := range d.Categories {
		if v := categories[cat]; v > 0 {
			det.Categories = append(det.Categories, d.CategoryShare{Category: cat, SalesCents: v})
		}
	}
	slices.SortFunc(det.Categories, func(a, b d.CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	// Last 6 calendar months, oldest first, zero-filled.
	now := s.now()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for k := 5; k >= 0; k-- {
		m := first.AddDate(0, -k, 0).Format("2006-01")
		det.Monthly = append(det.Monthly, d.MonthlySpend{Month: m, Cents: monthly[m]})
	}
	return det, nil
}

func (s *Store) AddCustomerNote(_ context.Context, customerID int64, author, text string) (d.CustomerNote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ci := s.customerIndex(customerID)
	if ci < 0 {
		return d.CustomerNote{}, d.ErrNotFound
	}
	n := d.CustomerNote{ID: s.nextNoteID, Author: author, Text: text, At: s.now()}
	s.nextNoteID++
	s.customers[ci].Notes = append(s.customers[ci].Notes, n)
	return n, nil
}

func (s *Store) customerIndex(id int64) int {
	return slices.IndexFunc(s.customers, func(c d.Customer) bool { return c.ID == id })
}

// recomputeCustomer derives order count, LTV, last order and segment from
// the customer's orders. Caller holds the write lock.
func (s *Store) recomputeCustomer(ci int, now time.Time) {
	c := &s.customers[ci]
	c.Orders, c.LTVCents, c.LastOrderAt = 0, 0, time.Time{}
	for _, o := range s.orders {
		if o.Customer.ID != c.ID {
			continue
		}
		if o.PlacedAt.After(c.LastOrderAt) {
			c.LastOrderAt = o.PlacedAt
		}
		if o.Status.Billable() {
			c.Orders++
			c.LTVCents += o.TotalCents
		}
	}
	switch {
	case !c.LastOrderAt.IsZero() && now.Sub(c.LastOrderAt) > 45*24*time.Hour:
		c.Segment = "At risk"
	case c.LTVCents >= 150_000 || c.Orders >= 8:
		c.Segment = "VIP"
	case c.Orders <= 1 || (!c.CreatedAt.IsZero() && now.Sub(c.CreatedAt) < 30*24*time.Hour):
		c.Segment = "New"
	default:
		c.Segment = "Regular"
	}
}

// ── Team ────────────────────────────────────────────────────────────────────

func (s *Store) ListMembers(_ context.Context, role d.Role) ([]d.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []d.Member{}
	for _, m := range s.members {
		if role == "" || m.Role == role {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Store) GetMember(_ context.Context, id int64) (d.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	return s.members[i], nil
}

func (s *Store) MemberByEmail(_ context.Context, email string) (d.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return strings.EqualFold(m.Email, email) })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	return s.members[i], nil
}

func (s *Store) TouchMember(_ context.Context, id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id }); i >= 0 {
		t := s.now()
		s.members[i].LastActiveAt = &t
	}
}

func (s *Store) CreateMember(_ context.Context, in d.MemberInput) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.members, func(m d.Member) bool { return strings.EqualFold(m.Email, in.Email) }) {
		return d.Member{}, d.NewValidationError("email", "is already a member")
	}
	m := d.Member{ID: s.nextMemberID, Name: in.Name, Email: in.Email, Role: in.Role, Status: d.MemberInvited}
	s.nextMemberID++
	s.members = append(s.members, m)
	return m, nil
}

func (s *Store) UpdateMember(_ context.Context, id int64, in d.MemberInput) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	if s.members[i].Role == d.RoleOwner {
		return d.Member{}, d.ErrForbidden
	}
	if slices.ContainsFunc(s.members, func(m d.Member) bool { return m.ID != id && strings.EqualFold(m.Email, in.Email) }) {
		return d.Member{}, d.NewValidationError("email", "is already a member")
	}
	s.members[i].Name, s.members[i].Email, s.members[i].Role = in.Name, in.Email, in.Role
	return s.members[i], nil
}

func (s *Store) DeleteMember(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.ErrNotFound
	}
	if s.members[i].Role == d.RoleOwner {
		return d.ErrForbidden
	}
	s.members = slices.Delete(s.members, i, i+1)
	return nil
}

// ── Dashboard ───────────────────────────────────────────────────────────────

const avgOrderCents = 8640

func (s *Store) Dashboard(_ context.Context, days int) (d.Dashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	days = min(max(days, 7), len(s.revenue))
	series := slices.Clone(s.revenue[len(s.revenue)-days:])
	var cur, prev int64
	for _, p := range series {
		cur += p.Current
		prev += p.Previous
	}

	orders := float64(cur) / avgOrderCents
	prevOrders := float64(prev) / avgOrderCents
	revenueTrend := bucket(series, 16)
	dash := d.Dashboard{
		RangeDays:        days,
		RevenueCents:     cur,
		PrevRevenueCents: prev,
		Revenue:          series,
		OrdersHeatmap:    s.heatmap,
		Activity:         s.activity,
		Markets:          markets[:7],
		KPIs: []d.KPI{
			{Key: "orders", Label: "Orders", Value: math.Round(orders), Unit: "count", DeltaPct: pct(orders, prevOrders), Trend: revenueTrend},
			{Key: "customers", Label: "New customers", Value: math.Round(orders * .62), Unit: "count", DeltaPct: pct(orders, prevOrders) * .55, Trend: wobble(revenueTrend, 1)},
			{Key: "conversion", Label: "Conversion", Value: 3.84, Unit: "percent", DeltaPct: -0.6, Trend: wobble(reverse(revenueTrend), 2)},
			{Key: "aov", Label: "Avg. order value", Value: avgOrderCents, Unit: "cents", DeltaPct: 2.1, Trend: wobble(revenueTrend, 3)},
		},
		Target: d.Target{Label: "Q4 target", BookedCents: 34_128_000, GoalCents: 120_000_000, PacePct: 6.2},
	}

	sales := map[d.Category]int64{}
	for _, p := range s.products {
		sales[p.Category] += p.Revenue30dCents()
	}
	for _, c := range d.Categories {
		dash.Categories = append(dash.Categories, d.CategoryShare{Category: c, SalesCents: sales[c]})
	}
	slices.SortFunc(dash.Categories, func(a, b d.CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	top := slices.Clone(s.products)
	slices.SortFunc(top, func(a, b d.Product) int { return cmp.Compare(b.Revenue30dCents(), a.Revenue30dCents()) })
	for _, p := range top[:min(5, len(top))] {
		dash.TopProducts = append(dash.TopProducts, d.TopProduct{ID: p.ID, Name: p.Name, Category: p.Category, Hue: p.Hue, ImageURL: p.ImageURL(), Sold: p.Sold30d, RevenueCents: p.Revenue30dCents()})
	}
	dash.RecentOrders = slices.Clone(s.orders[:min(6, len(s.orders))])
	return dash, nil
}

func pct(cur, prev float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Round((cur/prev-1)*1000) / 10
}

// bucket downsamples the revenue series to n averaged points.
func bucket(series []d.RevenuePoint, n int) []float64 {
	out := make([]float64, n)
	for i := range n {
		lo, hi := i*len(series)/n, max((i+1)*len(series)/n, i*len(series)/n+1)
		var sum float64
		for _, p := range series[lo:min(hi, len(series))] {
			sum += float64(p.Current)
		}
		out[i] = sum / float64(hi-lo)
	}
	return out
}

// wobble derives a differently-shaped but stable trend from another one.
func wobble(src []float64, k int) []float64 {
	out := make([]float64, len(src))
	for i, v := range src {
		out[i] = v * (1 + .12*math.Sin(float64(i*k)+float64(k)))
	}
	return out
}

func reverse(src []float64) []float64 {
	out := slices.Clone(src)
	slices.Reverse(out)
	return out
}
