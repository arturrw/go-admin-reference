// Package memory is an in-memory implementation of the admin store, seeded
// with deterministic fake data. It is the stand-in until the Postgres store
// (pgx + sqlc) lands; both satisfy httpapi.Store.
package memory

import (
	"bytes"
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
	activity       []d.Activity
	goals          map[string]d.TargetGoal // by quarter
	apiKeys        []d.APIKey              // newest first
	settings       d.Settings
	nextKeyID      int64
	nextCustomerID int64
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
		activity: ds.Activity, now: time.Now,
		goals: map[string]d.TargetGoal{}, nextKeyID: 1, settings: d.DefaultSettings(),
	}
	for _, p := range s.products {
		s.nextProductID = max(s.nextProductID, p.ID+1)
	}
	for _, m := range s.members {
		s.nextMemberID = max(s.nextMemberID, m.ID+1)
	}
	for _, c := range s.customers {
		s.nextCustomerID = max(s.nextCustomerID, c.ID+1)
		for _, n := range c.Notes {
			s.nextNoteID = max(s.nextNoteID, n.ID+1)
		}
	}
	// Seeded keys are listed newest first, like new ones.
	for _, k := range ds.APIKeys {
		k.ID = s.nextKeyID
		s.nextKeyID++
		s.apiKeys = slices.Insert(s.apiKeys, 0, k)
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
	d.ApplySales(out, s.salesByProduct())
	d.SortProducts(out, f.Sort)
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
	return s.view(s.products[i]), nil
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
	return s.view(p), nil
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
	return s.view(*p), nil
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
	return s.view(*p), nil
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
	return s.view(*p), img, nil
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
	return s.view(*p), nil
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

// view is a product as callers see it: copied, with its sales from orders.
func (s *Store) view(p d.Product) d.Product {
	one := []d.Product{cloneProduct(p)}
	d.ApplySales(one, s.salesByProduct())
	return one[0]
}

// salesByProduct tallies the last 30 days of orders; callers hold s.mu.
func (s *Store) salesByProduct() map[int64]*d.ProductSales {
	now := s.now()
	since := d.SalesWindowStart(now)
	var sales []d.Sale
	for _, o := range s.orders {
		if !o.PlacedAt.Before(since) {
			sales = append(sales, d.Sale{PlacedAt: o.PlacedAt, Status: o.Status, TotalCents: o.TotalCents, Items: o.Items})
		}
	}
	return d.SalesByProduct(now, sales)
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

func (s *Store) UpdateOrderStatus(_ context.Context, id int64, status d.OrderStatus, refund *d.OrderRefund, by string) (d.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.orders {
		if s.orders[i].ID == id {
			if s.orders[i].Status != status {
				s.orders[i].Events = append(slices.Clone(s.orders[i].Events), d.OrderEvent{Status: status, At: s.now(), By: by})
			}
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

func (s *Store) CreateCustomer(_ context.Context, in d.CustomerInput) (d.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.customers, func(c d.Customer) bool { return strings.EqualFold(c.Email, in.Email) }) {
		return d.Customer{}, d.NewValidationError("email", "is already a customer")
	}
	now := s.now()
	c := d.Customer{
		ID: s.nextCustomerID, Name: in.Name, Email: in.Email, Phone: in.Phone, Country: in.Country, Address: in.Address,
		Tags: append([]string{}, in.Tags...), AcceptsMarketing: in.AcceptsMarketing, Source: in.Source, Notes: []d.CustomerNote{},
		LastSeenAt: now, CreatedAt: now,
	}
	s.nextCustomerID++
	d.DeriveCustomer(&c, s.orders, now) // segment of a customer without orders: New
	s.customers = append(s.customers, c)
	return c, nil
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
			out = append(out, cloneMember(m))
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
	return cloneMember(s.members[i]), nil
}

func (s *Store) MemberByEmail(_ context.Context, email string) (d.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return strings.EqualFold(m.Email, email) })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	return cloneMember(s.members[i]), nil
}

func (s *Store) SetInvite(_ context.Context, id int64, hash []byte, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id && m.Status == d.MemberInvited })
	if i < 0 {
		return d.ErrNotFound
	}
	s.members[i].InviteHash, s.members[i].InviteExpiresAt = slices.Clone(hash), &expires
	return nil
}

func (s *Store) MemberByInvite(_ context.Context, hash []byte) (d.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.members {
		if m.Status == d.MemberInvited && m.InviteHash != nil && bytes.Equal(m.InviteHash, hash) && m.InviteExpiresAt.After(s.now()) {
			return cloneMember(m), nil
		}
	}
	return d.Member{}, d.ErrNotFound
}

func (s *Store) AcceptInvite(_ context.Context, id int64, name, passwordHash string) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id && m.Status == d.MemberInvited })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	now := s.now()
	m := &s.members[i]
	m.Name, m.PasswordHash, m.Status, m.InviteHash, m.InviteExpiresAt, m.LastActiveAt = name, passwordHash, d.MemberActive, nil, nil, &now
	return cloneMember(*m), nil
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
	return cloneMember(m), nil
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
	return cloneMember(s.members[i]), nil
}

func (s *Store) SetMemberAccess(_ context.Context, id int64, a d.MemberAccess) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	s.members[i].Granted, s.members[i].Revoked = slices.Clone(a.Granted), slices.Clone(a.Revoked)
	return cloneMember(s.members[i]), nil
}

func (s *Store) SetMemberStatus(_ context.Context, id int64, status d.MemberStatus) (d.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.Member{}, d.ErrNotFound
	}
	s.members[i].Status = status
	return cloneMember(s.members[i]), nil
}

// cloneMember copies the permission slices so callers cannot mutate store state.
func cloneMember(m d.Member) d.Member {
	m.Granted, m.Revoked = append([]d.Permission{}, m.Granted...), append([]d.Permission{}, m.Revoked...)
	return m
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

// ── Targets ─────────────────────────────────────────────────────────────────

func (s *Store) TargetGoal(_ context.Context, quarter string) (*d.TargetGoal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if g, ok := s.goals[quarter]; ok {
		return &g, nil
	}
	return nil, nil
}

func (s *Store) SetTargetGoal(_ context.Context, g d.TargetGoal) (d.TargetGoal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g.UpdatedAt = s.now()
	s.goals[g.Quarter] = g
	return g, nil
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
			(!f.ExcludeAuth || (a.Kind != d.ActAuth && a.Kind != d.ActAlert)) &&
			(f.NotifyMember == 0 || notifiesMember(a, f.NotifyMember)) &&
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

	now := s.now()
	days = d.DashboardDays(days)
	since := d.SalesSince(now, days)
	in := d.DashboardInput{Now: now, Days: days}
	for _, o := range s.orders {
		if !o.PlacedAt.Before(since) {
			in.Sales = append(in.Sales, d.Sale{PlacedAt: o.PlacedAt, Status: o.Status, TotalCents: o.TotalCents, Country: o.Customer.Country, Items: o.Items})
		}
	}
	for _, c := range s.customers {
		if !c.CreatedAt.Before(since) {
			in.CustomersSince = append(in.CustomersSince, c.CreatedAt)
		}
	}
	key, _, _ := d.QuarterOf(now)
	if g, ok := s.goals[key]; ok {
		in.Goal = &g
	}
	dash := d.BuildDashboard(in)
	dash.Activity, _ = s.listActivity(d.ActivityFilter{ExcludeAuth: true, Limit: d.DashboardActivity})
	dash.RecentOrders = s.withCovers(s.orders[:min(6, len(s.orders))]...)
	return dash, nil
}

// notifiesMember mirrors the NotifyMember condition of the Postgres query.
func notifiesMember(a d.Activity, member int64) bool {
	if a.Kind == d.ActAuth || a.ActorID == member {
		return false
	}
	return a.Kind != d.ActAlert || (a.Entity == "member" && a.EntityID == member)
}

func (s *Store) MarkNotificationsRead(_ context.Context, id int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.members, func(m d.Member) bool { return m.ID == id })
	if i < 0 {
		return d.ErrNotFound
	}
	s.members[i].NotificationsReadAt = &at
	return nil
}
