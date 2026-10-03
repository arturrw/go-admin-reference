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
	orders        []d.Order
	customers     []d.Customer
	members       []d.Member
	revenue       []d.RevenuePoint
	heatmap       [7][24]int
	activity      []d.Activity
	nextProductID int64
	nextMemberID  int64
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
			contains(p.Name+" "+p.SKU, f.Query) {
			out = append(out, p)
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
	return s.products[i], nil
}

func (s *Store) CreateProduct(_ context.Context, in d.ProductInput) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := d.Product{
		ID: s.nextProductID, Hue: categoryHue(in.Category), Trend: make([]int, 14), UpdatedAt: s.now(),
	}
	applyProductInput(&p, in)
	s.nextProductID++
	s.products = append([]d.Product{p}, s.products...)
	return p, nil
}

func (s *Store) UpdateProduct(_ context.Context, id int64, in d.ProductInput) (d.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(id)
	if i < 0 {
		return d.Product{}, d.ErrNotFound
	}
	p := &s.products[i]
	if p.Category != in.Category {
		p.Hue = categoryHue(in.Category)
	}
	applyProductInput(p, in)
	p.UpdatedAt = s.now()
	return *p, nil
}

func (s *Store) DeleteProduct(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.productIndex(id)
	if i < 0 {
		return d.ErrNotFound
	}
	s.products = slices.Delete(s.products, i, i+1)
	return nil
}

func (s *Store) BulkProducts(_ context.Context, ids []int64, action d.BulkAction) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	if action == d.BulkDelete {
		s.products = slices.DeleteFunc(s.products, func(p d.Product) bool {
			hit := slices.Contains(ids, p.ID)
			if hit {
				n++
			}
			return hit
		})
		return n, nil
	}
	status := d.ProductActive
	if action == d.BulkArchive {
		status = d.ProductArchived
	}
	for i := range s.products {
		if slices.Contains(ids, s.products[i].ID) {
			s.products[i].Status = status
			s.products[i].UpdatedAt = s.now()
			n++
		}
	}
	return n, nil
}

func (s *Store) productIndex(id int64) int {
	return slices.IndexFunc(s.products, func(p d.Product) bool { return p.ID == id })
}

func applyProductInput(p *d.Product, in d.ProductInput) {
	p.Name, p.SKU, p.Category, p.PriceCents = in.Name, in.SKU, in.Category, in.PriceCents
	p.Stock, p.Status, p.Description = in.Stock, in.Status, in.Description
}

func categoryHue(c d.Category) int { return catalog[c].hue }

// ── Orders ──────────────────────────────────────────────────────────────────

func (s *Store) ListOrders(_ context.Context, f d.OrderFilter) ([]d.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []d.Order{}
	for _, o := range s.orders {
		if (f.Status == "" || o.Status == f.Status) && (contains(o.Customer.Name, f.Query) || contains("#"+strconv.FormatInt(o.ID, 10), f.Query)) {
			out = append(out, o)
			if f.Limit > 0 && len(out) == f.Limit {
				break
			}
		}
	}
	return out, nil
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
		if (f.Segment == "" || c.Segment == f.Segment) && contains(c.Name+" "+c.Email, f.Query) {
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

func (s *Store) CreateMember(_ context.Context, in d.MemberInput) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.members, func(m d.Member) bool { return m.Email == in.Email }) {
		return d.Member{}, &d.ValidationError{Fields: map[string]string{"email": "is already a member"}}
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
		dash.TopProducts = append(dash.TopProducts, d.TopProduct{ID: p.ID, Name: p.Name, Category: p.Category, Hue: p.Hue, Sold: p.Sold30d, RevenueCents: p.Revenue30dCents()})
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
