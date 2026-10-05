// Package memory is an in-memory implementation of the admin store, seeded
// with deterministic fake data. It is the stand-in until the Postgres store
// (pgx + sqlc) lands; both satisfy httpapi.Store.
package memory

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/seed"
)

type Store struct {
	mu             sync.RWMutex
	products       []d.Product
	orders         []d.Order // newest first
	customers      []d.Customer
	members        []d.Member
	revenue        []d.RevenuePoint
	heatmap        [7][24]int
	activity       []d.Activity
	nextProductID  int64
	nextMemberID   int64
	nextNoteID     int64
	nextActivityID int64
	now            func() time.Time
}

func New(now time.Time) *Store {
	ds := seed.Generate(now)
	s := &Store{
		products: ds.Products, orders: ds.Orders, customers: ds.Customers, members: ds.Members,
		revenue: ds.Revenue, heatmap: ds.Heatmap, activity: ds.Activity, now: time.Now,
	}
	for _, p := range s.products {
		s.nextProductID = max(s.nextProductID, p.ID+1)
	}
	for _, m := range s.members {
		s.nextMemberID = max(s.nextMemberID, m.ID+1)
	}
	for _, c := range s.customers {
		for _, n := range c.Notes {
			s.nextNoteID = max(s.nextNoteID, n.ID+1)
		}
	}
	slices.SortFunc(s.activity, func(a, b d.Activity) int { return cmp.Or(b.At.Compare(a.At), cmp.Compare(b.ID, a.ID)) })
	for _, a := range s.activity {
		s.nextActivityID = max(s.nextActivityID, a.ID+1)
	}
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

func categoryHue(c d.Category) int { return seed.CategoryHue(c) }

// ── Orders ──────────────────────────────────────────────────────────────────

// withCovers copies orders and points each item's image at the product's
// current cover, so orders follow gallery edits. Items of deleted products
// keep the snapshot taken at purchase.
func (s *Store) withCovers(orders ...d.Order) []d.Order {
	out := make([]d.Order, len(orders))
	for i, o := range orders {
		o.Items = slices.Clone(o.Items)
		for j, it := range o.Items {
			if pi := s.productIndex(it.ProductID); pi >= 0 {
				o.Items[j].ImageURL = s.products[pi].ImageURL()
			}
		}
		out[i] = o
	}
	return out
}

func (s *Store) ListOrders(_ context.Context, f d.OrderFilter) ([]d.Order, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matched []d.Order
	for _, o := range s.orders {
		if (f.Status == "" || o.Status == f.Status) &&
			(f.CustomerID == 0 || o.Customer.ID == f.CustomerID) &&
			(f.From.IsZero() || !o.PlacedAt.Before(f.From)) &&
			(f.To.IsZero() || o.PlacedAt.Before(f.To)) &&
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
	return s.withCovers(matched[lo:hi]...), total, nil
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
			return s.withCovers(o)[0], nil
		}
	}
	return d.Order{}, d.ErrNotFound
}

func (s *Store) UpdateOrderStatus(_ context.Context, id int64, status d.OrderStatus, refund *d.OrderRefund) (d.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID == id {
			s.orders[i].Status = status
			s.orders[i].Refund = refund
			if ci := s.customerIndex(s.orders[i].Customer.ID); ci >= 0 {
				s.recomputeCustomer(ci, s.now())
			}
			return s.withCovers(s.orders[i])[0], nil
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
	var orders []d.Order
	for _, o := range s.orders {
		if o.Customer.ID == id {
			orders = append(orders, o)
		}
	}
	return d.BuildCustomerDetail(c, s.withCovers(orders...), s.now()), nil
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

func (s *Store) DeleteCustomerNote(_ context.Context, customerID, noteID int64) (d.CustomerNote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ci := s.customerIndex(customerID)
	if ci < 0 {
		return d.CustomerNote{}, d.ErrNotFound
	}
	notes := s.customers[ci].Notes
	i := slices.IndexFunc(notes, func(n d.CustomerNote) bool { return n.ID == noteID })
	if i < 0 {
		return d.CustomerNote{}, d.ErrNotFound
	}
	n := notes[i]
	s.customers[ci].Notes = slices.Delete(slices.Clone(notes), i, i+1)
	return n, nil
}

func (s *Store) customerIndex(id int64) int {
	return slices.IndexFunc(s.customers, func(c d.Customer) bool { return c.ID == id })
}

// recomputeCustomer refreshes derived customer fields. Caller holds the write lock.
func (s *Store) recomputeCustomer(ci int, now time.Time) {
	d.DeriveCustomer(&s.customers[ci], s.orders, now)
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

// ── Activity ────────────────────────────────────────────────────────────────

func (s *Store) RecordActivity(_ context.Context, a d.Activity) (d.Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = s.nextActivityID
	a.At = s.now()
	s.nextActivityID++
	s.activity = slices.Insert(s.activity, 0, a)
	return a, nil
}

func (s *Store) ListActivity(_ context.Context, f d.ActivityFilter) ([]d.Activity, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out, total := s.listActivity(f)
	return out, total, nil
}

// listActivity filters the newest-first log. Caller holds a lock.
func (s *Store) listActivity(f d.ActivityFilter) ([]d.Activity, int) {
	var matched []d.Activity
	for _, a := range s.activity {
		if (f.ActorID == 0 || a.ActorID == f.ActorID) &&
			(f.Kind == "" || a.Kind == f.Kind) &&
			(!f.ExcludeAuth || a.Kind != d.ActAuth) &&
			contains(a.Actor+" "+a.Message, f.Query) {
			matched = append(matched, a)
		}
	}
	total := len(matched)
	lo := min(max(f.Offset, 0), total)
	hi := total
	if f.Limit > 0 {
		hi = min(lo+f.Limit, total)
	}
	return append([]d.Activity{}, matched[lo:hi]...), total
}

// ── Dashboard ───────────────────────────────────────────────────────────────

func (s *Store) Dashboard(_ context.Context, days int) (d.Dashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	days = min(max(days, 7), len(s.revenue))
	dash := d.BuildDashboard(slices.Clone(s.revenue[len(s.revenue)-days:]), seed.Markets())
	dash.OrdersHeatmap = s.heatmap
	dash.Activity, _ = s.listActivity(d.ActivityFilter{ExcludeAuth: true, Limit: d.DashboardActivity})

	sales := map[d.Category]int64{}
	for _, p := range s.products {
		sales[p.Category] += p.Revenue30dCents()
	}
	for _, c := range d.Categories {
		dash.Categories = append(dash.Categories, d.CategoryShare{Category: c, SalesCents: sales[c]})
	}
	slices.SortFunc(dash.Categories, func(a, b d.CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	top := slices.Clone(s.products)
	slices.SortFunc(top, func(a, b d.Product) int {
		return cmp.Or(cmp.Compare(b.Revenue30dCents(), a.Revenue30dCents()), cmp.Compare(a.ID, b.ID))
	})
	for _, p := range top[:min(5, len(top))] {
		dash.TopProducts = append(dash.TopProducts, d.TopProduct{ID: p.ID, Name: p.Name, Category: p.Category, Hue: p.Hue, ImageURL: p.ImageURL(), Sold: p.Sold30d, RevenueCents: p.Revenue30dCents()})
	}
	dash.RecentOrders = s.withCovers(s.orders[:min(6, len(s.orders))]...)
	return dash, nil
}
