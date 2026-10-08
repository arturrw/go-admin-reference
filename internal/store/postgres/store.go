package postgres

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"math"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/seed"
	"github.com/arturrw/go-admin-reference/internal/store/postgres/db"
)

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// ── Products ────────────────────────────────────────────────────────────────

func toImage(i db.ProductImage) d.ProductImage {
	return d.ProductImage{ID: i.ID, URL: i.Url, Alt: i.Alt, Generated: i.Generated, SizeBytes: i.SizeBytes}
}

func toProduct(p db.Product, imgs []db.ProductImage) d.Product {
	out := d.Product{
		ID: p.ID, Name: p.Name, SKU: p.Sku, Category: d.Category(p.Category), Vendor: p.Vendor, Tags: p.Tags,
		PriceCents: p.PriceCents, CompareAtCents: p.CompareAtCents, CostCents: p.CostCents,
		Stock: int(p.Stock), WeightGrams: int(p.WeightGrams), Status: d.ProductStatus(p.Status),
		Sold30d: int(p.Sold30d), Rating: math.Round(float64(p.Rating)*10) / 10, Hue: int(p.Hue),
		Description: p.Description, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		Trend: make([]int, len(p.Trend)), Images: make([]d.ProductImage, 0, len(imgs)),
	}
	for i, v := range p.Trend {
		out.Trend[i] = int(v)
	}
	for _, img := range imgs {
		out.Images = append(out.Images, toImage(img))
	}
	return out
}

// withImages loads galleries for a batch of products in one query.
func (s *Store) withImages(ctx context.Context, q *db.Queries, rows []db.Product) ([]d.Product, error) {
	ids := make([]int64, len(rows))
	for i, p := range rows {
		ids[i] = p.ID
	}
	imgs, err := q.ListProductImages(ctx, ids)
	if err != nil {
		return nil, err
	}
	byProduct := map[int64][]db.ProductImage{}
	for _, img := range imgs {
		byProduct[img.ProductID] = append(byProduct[img.ProductID], img)
	}
	out := make([]d.Product, len(rows))
	for i, p := range rows {
		out[i] = toProduct(p, byProduct[p.ID])
	}
	return out, nil
}

func (s *Store) ListProducts(ctx context.Context, f d.ProductFilter) ([]d.Product, error) {
	rows, err := s.q.ListProducts(ctx, db.ListProductsParams{
		Category: string(f.Category), Status: string(f.Status), Q: escapeLike(f.Query), Sort: f.Sort,
	})
	if err != nil {
		return nil, err
	}
	return s.withImages(ctx, s.q, rows)
}

func (s *Store) ProductStats(ctx context.Context) (d.ProductStats, error) {
	row, err := s.q.ProductStats(ctx)
	if err != nil {
		return d.ProductStats{}, err
	}
	counts, err := s.q.ProductCountsByCategory(ctx)
	if err != nil {
		return d.ProductStats{}, err
	}
	st := d.ProductStats{
		Total: int(row.Total), Active: int(row.Active), LowStock: int(row.LowStock), OutOfStock: int(row.OutOfStock),
		InventoryValueCents: row.InventoryValueCents, ByCategory: map[d.Category]int{},
	}
	for _, c := range counts {
		st.ByCategory[d.Category(c.Category)] = int(c.N)
	}
	return st, nil
}

func (s *Store) getProduct(ctx context.Context, q *db.Queries, id int64) (d.Product, error) {
	p, err := q.GetProduct(ctx, id)
	if err != nil {
		return d.Product{}, mapErr(err)
	}
	out, err := s.withImages(ctx, q, []db.Product{p})
	if err != nil {
		return d.Product{}, err
	}
	return out[0], nil
}

func (s *Store) GetProduct(ctx context.Context, id int64) (d.Product, error) {
	return s.getProduct(ctx, s.q, id)
}

func (s *Store) CreateProduct(ctx context.Context, in d.ProductInput) (d.Product, error) {
	p, err := s.q.CreateProduct(ctx, db.CreateProductParams{
		Name: in.Name, Sku: in.SKU, Category: string(in.Category), Vendor: in.Vendor, Tags: nonNil(in.Tags),
		PriceCents: in.PriceCents, CompareAtCents: in.CompareAtCents, CostCents: in.CostCents,
		Stock: int32(in.Stock), WeightGrams: int32(in.WeightGrams), Status: string(in.Status),
		Description: in.Description, Hue: int32(seed.CategoryHue(in.Category)),
	})
	if err != nil {
		return d.Product{}, mapErr(err)
	}
	return toProduct(p, nil), nil
}

func (s *Store) UpdateProduct(ctx context.Context, id int64, in d.ProductInput) (d.Product, error) {
	_, err := s.q.UpdateProduct(ctx, db.UpdateProductParams{
		ID: id, Name: in.Name, Sku: in.SKU, Category: string(in.Category), Hue: int32(seed.CategoryHue(in.Category)),
		Vendor: in.Vendor, Tags: nonNil(in.Tags), PriceCents: in.PriceCents, CompareAtCents: in.CompareAtCents,
		CostCents: in.CostCents, Stock: int32(in.Stock), WeightGrams: int32(in.WeightGrams),
		Status: string(in.Status), Description: in.Description,
	})
	if err != nil {
		return d.Product{}, mapErr(err)
	}
	return s.GetProduct(ctx, id)
}

// DeleteProduct returns the deleted product so the caller can remove files.
func (s *Store) DeleteProduct(ctx context.Context, id int64) (d.Product, error) {
	var p d.Product
	err := s.tx(ctx, func(q *db.Queries) error {
		var err error
		if p, err = s.getProduct(ctx, q, id); err != nil {
			return err
		}
		_, err = q.DeleteProduct(ctx, id)
		return err
	})
	return p, mapErr(err)
}

func (s *Store) BulkProducts(ctx context.Context, ids []int64, action d.BulkAction) ([]d.Product, error) {
	var affected []d.Product
	err := s.tx(ctx, func(q *db.Queries) error {
		var rows []db.Product
		for _, id := range ids {
			p, err := q.GetProduct(ctx, id)
			if err == nil {
				rows = append(rows, p)
			}
		}
		var err error
		if affected, err = s.withImages(ctx, q, rows); err != nil {
			return err
		}
		switch action {
		case d.BulkDelete:
			_, err = q.DeleteProducts(ctx, ids)
		case d.BulkArchive:
			_, err = q.SetProductsStatus(ctx, db.SetProductsStatusParams{Status: string(d.ProductArchived), Ids: ids})
		default:
			_, err = q.SetProductsStatus(ctx, db.SetProductsStatusParams{Status: string(d.ProductActive), Ids: ids})
		}
		return err
	})
	return affected, mapErr(err)
}

func (s *Store) AddProductImage(ctx context.Context, productID int64, img d.ProductImage) (d.Product, error) {
	var out d.Product
	err := s.tx(ctx, func(q *db.Queries) error {
		if _, err := q.GetProduct(ctx, productID); err != nil {
			return err
		}
		n, err := q.CountProductImages(ctx, productID)
		if err != nil {
			return err
		}
		if n >= d.MaxProductImages {
			return d.NewValidationError("file", "a product can have at most 8 images")
		}
		if err := q.AddProductImage(ctx, db.AddProductImageParams{
			ID: img.ID, ProductID: productID, Url: img.URL, Alt: img.Alt, Generated: img.Generated, SizeBytes: img.SizeBytes,
		}); err != nil {
			return err
		}
		if err := q.TouchProduct(ctx, productID); err != nil {
			return err
		}
		out, err = s.getProduct(ctx, q, productID)
		return err
	})
	return out, mapErr(err)
}

func (s *Store) DeleteProductImage(ctx context.Context, productID int64, imageID string) (d.Product, d.ProductImage, error) {
	var (
		out d.Product
		img db.ProductImage
	)
	err := s.tx(ctx, func(q *db.Queries) error {
		var err error
		if img, err = q.DeleteProductImage(ctx, db.DeleteProductImageParams{ProductID: productID, ID: imageID}); err != nil {
			return err
		}
		if err = q.TouchProduct(ctx, productID); err != nil {
			return err
		}
		out, err = s.getProduct(ctx, q, productID)
		return err
	})
	return out, toImage(img), mapErr(err)
}

func (s *Store) SetPrimaryImage(ctx context.Context, productID int64, imageID string) (d.Product, error) {
	n, err := s.q.SetPrimaryImage(ctx, db.SetPrimaryImageParams{ProductID: productID, ImageID: imageID})
	if err != nil {
		return d.Product{}, err
	}
	if n == 0 {
		return d.Product{}, d.ErrNotFound
	}
	_ = s.q.TouchProduct(ctx, productID)
	return s.GetProduct(ctx, productID)
}

// ── Orders ──────────────────────────────────────────────────────────────────

func (s *Store) attachItems(ctx context.Context, orders []d.Order) ([]d.Order, error) {
	ids := make([]int64, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	items, err := s.q.ListOrderItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	byOrder := map[int64][]d.OrderItem{}
	for _, it := range items {
		byOrder[it.OrderID] = append(byOrder[it.OrderID], d.OrderItem{
			ProductID: it.ProductID.Int64, Name: it.Name, SKU: it.Sku, Category: d.Category(it.Category),
			Hue: int(it.Hue), ImageURL: it.ImageUrl, Qty: int(it.Qty), PriceCents: it.PriceCents,
		})
	}
	for i := range orders {
		orders[i].Items = nonNil(byOrder[orders[i].ID])
	}
	return orders, nil
}

func listRowToOrder(r db.ListOrdersRow) d.Order {
	o := d.Order{
		ID: r.ID, Status: d.OrderStatus(r.Status), Payment: r.Payment, TotalCents: r.TotalCents, PlacedAt: r.PlacedAt,
		Customer: d.CustomerRef{ID: r.CustomerID, Name: r.CustomerName, Email: r.CustomerEmail, Country: r.CustomerCountry, Segment: r.CustomerSegment},
	}
	if r.RefundedAt != nil {
		o.Refund = &d.OrderRefund{Reason: r.RefundReason, By: r.RefundedBy, At: *r.RefundedAt}
	}
	return o
}

func (s *Store) ListOrders(ctx context.Context, f d.OrderFilter) ([]d.Order, int, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}
	q := escapeLike(f.Query)
	from, to := timePtr(f.From), timePtr(f.To)
	rows, err := s.q.ListOrders(ctx, db.ListOrdersParams{
		Status: string(f.Status), CustomerID: f.CustomerID, PlacedFrom: from, PlacedTo: to, Q: q,
		Lim: int32(min(limit, math.MaxInt32)), Off: int32(max(f.Offset, 0)),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountOrders(ctx, db.CountOrdersParams{Status: string(f.Status), CustomerID: f.CustomerID, PlacedFrom: from, PlacedTo: to, Q: q})
	if err != nil {
		return nil, 0, err
	}
	orders := make([]d.Order, len(rows))
	for i, r := range rows {
		orders[i] = listRowToOrder(r)
	}
	orders, err = s.attachItems(ctx, orders)
	return orders, int(total), err
}

func (s *Store) OrderCounts(ctx context.Context) (map[d.OrderStatus]int, error) {
	rows, err := s.q.OrderCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := map[d.OrderStatus]int{}
	for _, r := range rows {
		out[d.OrderStatus(r.Status)] = int(r.N)
	}
	return out, nil
}

func (s *Store) GetOrder(ctx context.Context, id int64) (d.Order, error) {
	r, err := s.q.GetOrder(ctx, id)
	if err != nil {
		return d.Order{}, mapErr(err)
	}
	orders, err := s.attachItems(ctx, []d.Order{listRowToOrder(db.ListOrdersRow(r))})
	if err != nil {
		return d.Order{}, err
	}
	return orders[0], nil
}

func (s *Store) UpdateOrderStatus(ctx context.Context, id int64, status d.OrderStatus, refund *d.OrderRefund) (d.Order, error) {
	p := db.UpdateOrderStatusParams{ID: id, Status: string(status)}
	if refund != nil {
		p.RefundReason, p.RefundedBy, p.RefundedAt = refund.Reason, refund.By, &refund.At
	}
	n, err := s.q.UpdateOrderStatus(ctx, p)
	if err != nil {
		return d.Order{}, mapErr(err)
	}
	if n == 0 {
		return d.Order{}, d.ErrNotFound
	}
	return s.GetOrder(ctx, id)
}

// ── Customers ───────────────────────────────────────────────────────────────

func toCustomer(c db.CustomersV) d.Customer {
	out := d.Customer{
		ID: c.ID, Name: c.Name, Email: c.Email, Phone: c.Phone, Country: c.Country,
		Address: d.Address{Line1: c.AddressLine1, City: c.City, PostalCode: c.PostalCode, Country: c.Country},
		Orders:  int(c.Orders), LTVCents: c.LtvCents, Segment: c.Segment, Tags: nonNil(c.Tags),
		AcceptsMarketing: c.AcceptsMarketing, Source: c.Source, Notes: []d.CustomerNote{},
		LastSeenAt: c.LastSeenAt, CreatedAt: c.CreatedAt,
	}
	if c.LastOrderAt != nil {
		out.LastOrderAt = *c.LastOrderAt
	}
	return out
}

func (s *Store) ListCustomers(ctx context.Context, f d.CustomerFilter) ([]d.Customer, error) {
	rows, err := s.q.ListCustomers(ctx, db.ListCustomersParams{Segment: f.Segment, Q: escapeLike(f.Query)})
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(rows))
	out := make([]d.Customer, len(rows))
	index := map[int64]int{}
	for i, r := range rows {
		ids[i], out[i], index[r.ID] = r.ID, toCustomer(r), i
	}
	notes, err := s.q.ListCustomerNotes(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		i := index[n.CustomerID]
		out[i].Notes = append(out[i].Notes, d.CustomerNote{ID: n.ID, Author: n.Author, Text: n.Body, At: n.CreatedAt})
	}
	return out, nil
}

func (s *Store) CustomerSegments(ctx context.Context) (map[string]d.SegmentSummary, error) {
	rows, err := s.q.CustomerSegments(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]d.SegmentSummary{}
	for _, r := range rows {
		out[r.Segment] = d.SegmentSummary{Count: int(r.N), LTVCents: r.LtvCents}
	}
	return out, nil
}

func (s *Store) GetCustomer(ctx context.Context, id int64) (d.CustomerDetail, error) {
	row, err := s.q.GetCustomer(ctx, id)
	if err != nil {
		return d.CustomerDetail{}, mapErr(err)
	}
	c := toCustomer(row)
	notes, err := s.q.ListCustomerNotes(ctx, []int64{id})
	if err != nil {
		return d.CustomerDetail{}, err
	}
	for _, n := range notes {
		c.Notes = append(c.Notes, d.CustomerNote{ID: n.ID, Author: n.Author, Text: n.Body, At: n.CreatedAt})
	}
	orders, _, err := s.ListOrders(ctx, d.OrderFilter{CustomerID: id})
	if err != nil {
		return d.CustomerDetail{}, err
	}
	return d.BuildCustomerDetail(c, orders, s.now()), nil
}

func (s *Store) CreateCustomer(ctx context.Context, in d.CustomerInput) (d.Customer, error) {
	id, err := s.q.CreateCustomer(ctx, db.CreateCustomerParams{
		Name: in.Name, Email: in.Email, Phone: in.Phone, Country: in.Country, AddressLine1: in.Address.Line1, City: in.Address.City,
		PostalCode: in.Address.PostalCode, Tags: nonNil(in.Tags), AcceptsMarketing: in.AcceptsMarketing, Source: in.Source,
	})
	if err != nil {
		return d.Customer{}, mapErr(err)
	}
	det, err := s.GetCustomer(ctx, id)
	return det.Customer, err
}

func (s *Store) AddCustomerNote(ctx context.Context, customerID int64, author, text string) (d.CustomerNote, error) {
	n, err := s.q.AddCustomerNote(ctx, db.AddCustomerNoteParams{CustomerID: customerID, Author: author, Body: text})
	if err != nil {
		return d.CustomerNote{}, mapErr(err) // FK violation → not found
	}
	return d.CustomerNote{ID: n.ID, Author: n.Author, Text: n.Body, At: n.CreatedAt}, nil
}

func (s *Store) DeleteCustomerNote(ctx context.Context, customerID, noteID int64) (d.CustomerNote, error) {
	n, err := s.q.DeleteCustomerNote(ctx, db.DeleteCustomerNoteParams{ID: noteID, CustomerID: customerID})
	if err != nil {
		return d.CustomerNote{}, mapErr(err) // no row → not found
	}
	return d.CustomerNote{ID: n.ID, Author: n.Author, Text: n.Body, At: n.CreatedAt}, nil
}

// ── Team ────────────────────────────────────────────────────────────────────

func toMember(m db.Member) d.Member {
	return d.Member{
		ID: m.ID, Name: m.Name, Email: m.Email, Role: d.Role(m.Role), Status: d.MemberStatus(m.Status),
		MFA: m.Mfa, LastActiveAt: m.LastActiveAt, PasswordHash: m.PasswordHash, NotificationsReadAt: m.NotificationsReadAt, InviteHash: m.InviteHash, InviteExpiresAt: m.InviteExpiresAt,
		Granted: toPerms(m.Granted), Revoked: toPerms(m.Revoked),
	}
}

func toPerms(ss []string) []d.Permission {
	out := make([]d.Permission, len(ss))
	for i, s := range ss {
		out[i] = d.Permission(s)
	}
	return out
}

func fromPerms(ps []d.Permission) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = string(p)
	}
	return out
}

func (s *Store) SetMemberAccess(ctx context.Context, id int64, a d.MemberAccess) (d.Member, error) {
	m, err := s.q.SetMemberAccess(ctx, db.SetMemberAccessParams{ID: id, Granted: fromPerms(a.Granted), Revoked: fromPerms(a.Revoked)})
	return toMember(m), mapErr(err)
}

func (s *Store) SetMemberStatus(ctx context.Context, id int64, status d.MemberStatus) (d.Member, error) {
	m, err := s.q.SetMemberStatus(ctx, db.SetMemberStatusParams{ID: id, Status: string(status)})
	return toMember(m), mapErr(err)
}

func (s *Store) ListMembers(ctx context.Context, role d.Role) ([]d.Member, error) {
	rows, err := s.q.ListMembers(ctx, string(role))
	if err != nil {
		return nil, err
	}
	out := make([]d.Member, len(rows))
	for i, m := range rows {
		out[i] = toMember(m)
	}
	return out, nil
}

func (s *Store) GetMember(ctx context.Context, id int64) (d.Member, error) {
	m, err := s.q.GetMember(ctx, id)
	return toMember(m), mapErr(err)
}

func (s *Store) MemberByEmail(ctx context.Context, email string) (d.Member, error) {
	m, err := s.q.MemberByEmail(ctx, email)
	return toMember(m), mapErr(err)
}

func (s *Store) MarkNotificationsRead(ctx context.Context, id int64, at time.Time) error {
	return s.q.MarkNotificationsRead(ctx, db.MarkNotificationsReadParams{ID: id, NotificationsReadAt: &at})
}

func (s *Store) SetInvite(ctx context.Context, id int64, hash []byte, expires time.Time) error {
	n, err := s.q.SetInvite(ctx, db.SetInviteParams{ID: id, InviteHash: hash, InviteExpiresAt: &expires})
	if err != nil {
		return mapErr(err)
	}
	if n == 0 {
		return d.ErrNotFound
	}
	return nil
}

func (s *Store) MemberByInvite(ctx context.Context, hash []byte) (d.Member, error) {
	m, err := s.q.MemberByInvite(ctx, hash)
	return toMember(m), mapErr(err)
}

func (s *Store) AcceptInvite(ctx context.Context, id int64, name, passwordHash string) (d.Member, error) {
	m, err := s.q.AcceptInvite(ctx, db.AcceptInviteParams{ID: id, Name: name, PasswordHash: passwordHash})
	return toMember(m), mapErr(err)
}

func (s *Store) TouchMember(ctx context.Context, id int64) { _ = s.q.TouchMember(ctx, id) }

func (s *Store) CreateMember(ctx context.Context, in d.MemberInput) (d.Member, error) {
	m, err := s.q.CreateMember(ctx, db.CreateMemberParams{Name: in.Name, Email: in.Email, Role: string(in.Role)})
	return toMember(m), mapErr(err)
}

func (s *Store) UpdateMember(ctx context.Context, id int64, in d.MemberInput) (d.Member, error) {
	var out d.Member
	err := s.tx(ctx, func(q *db.Queries) error {
		cur, err := q.GetMember(ctx, id)
		if err != nil {
			return err
		}
		if d.Role(cur.Role) == d.RoleOwner {
			return d.ErrForbidden
		}
		m, err := q.UpdateMember(ctx, db.UpdateMemberParams{ID: id, Name: in.Name, Email: in.Email, Role: string(in.Role)})
		out = toMember(m)
		return err
	})
	return out, mapErr(err)
}

func (s *Store) DeleteMember(ctx context.Context, id int64) error {
	return mapErr(s.tx(ctx, func(q *db.Queries) error {
		cur, err := q.GetMember(ctx, id)
		if err != nil {
			return err
		}
		if d.Role(cur.Role) == d.RoleOwner {
			return d.ErrForbidden
		}
		_, err = q.DeleteMember(ctx, id)
		return err
	}))
}

// ── Dashboard ───────────────────────────────────────────────────────────────

func (s *Store) Dashboard(ctx context.Context, days int) (d.Dashboard, error) {
	days = min(max(days, 7), 90)
	rev, err := s.q.RevenueSeries(ctx, int32(days))
	if err != nil {
		return d.Dashboard{}, err
	}
	series := make([]d.RevenuePoint, len(rev))
	for i, r := range rev {
		series[i] = d.RevenuePoint{Date: r.Day.Format("2006-01-02"), Current: r.CurrentCents, Previous: r.PreviousCents}
	}
	dash := d.BuildDashboard(series, seed.Markets())
	key, _, _ := d.QuarterOf(s.now())
	goal, err := s.TargetGoal(ctx, key)
	if err != nil {
		return d.Dashboard{}, err
	}
	dash.Target = d.BuildTarget(s.now(), goal)

	cells, err := s.q.OrdersHeatmap(ctx)
	if err != nil {
		return d.Dashboard{}, err
	}
	for _, c := range cells {
		dash.OrdersHeatmap[c.Dow][c.Hour] = int(c.Orders)
	}

	sales, err := s.q.ProductSalesByCategory(ctx)
	if err != nil {
		return d.Dashboard{}, err
	}
	byCat := map[d.Category]int64{}
	for _, r := range sales {
		byCat[d.Category(r.Category)] = r.SalesCents
	}
	for _, c := range d.Categories {
		dash.Categories = append(dash.Categories, d.CategoryShare{Category: c, SalesCents: byCat[c]})
	}
	slices.SortFunc(dash.Categories, func(a, b d.CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	topRows, err := s.q.TopProducts(ctx, 5)
	if err != nil {
		return d.Dashboard{}, err
	}
	top, err := s.withImages(ctx, s.q, topRows)
	if err != nil {
		return d.Dashboard{}, err
	}
	for _, p := range top {
		dash.TopProducts = append(dash.TopProducts, d.TopProduct{ID: p.ID, Name: p.Name, Category: p.Category, Hue: p.Hue, ImageURL: p.ImageURL(), Sold: p.Sold30d, RevenueCents: p.Revenue30dCents()})
	}

	if dash.RecentOrders, _, err = s.ListOrders(ctx, d.OrderFilter{Limit: 6}); err != nil {
		return d.Dashboard{}, err
	}
	if dash.Activity, _, err = s.ListActivity(ctx, d.ActivityFilter{ExcludeAuth: true, Limit: d.DashboardActivity}); err != nil {
		return d.Dashboard{}, err
	}
	return dash, nil
}

// TargetGoal returns the stored goal for a quarter, or nil when none is set.
func (s *Store) TargetGoal(ctx context.Context, quarter string) (*d.TargetGoal, error) {
	t, err := s.q.GetTarget(ctx, quarter)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d.TargetGoal{Quarter: t.Quarter, GoalCents: t.GoalCents, UpdatedBy: t.UpdatedBy, UpdatedAt: t.UpdatedAt}, nil
}

func (s *Store) SetTargetGoal(ctx context.Context, g d.TargetGoal) (d.TargetGoal, error) {
	t, err := s.q.UpsertTarget(ctx, db.UpsertTargetParams{Quarter: g.Quarter, GoalCents: g.GoalCents, UpdatedBy: g.UpdatedBy})
	if err != nil {
		return d.TargetGoal{}, mapErr(err)
	}
	return d.TargetGoal{Quarter: t.Quarter, GoalCents: t.GoalCents, UpdatedBy: t.UpdatedBy, UpdatedAt: t.UpdatedAt}, nil
}

// ── Activity ────────────────────────────────────────────────────────────────

func toActivity(a db.Activity) d.Activity {
	return d.Activity{
		ID: a.ID, Kind: a.Kind, ActorID: a.ActorID.Int64, Actor: a.Actor, Message: a.Message,
		Entity: a.Entity, EntityID: a.EntityID, At: a.At,
	}
}

func (s *Store) RecordActivity(ctx context.Context, a d.Activity) (d.Activity, error) {
	row, err := s.q.AddActivity(ctx, db.AddActivityParams{
		Kind: a.Kind, Actor: a.Actor, ActorID: pgtype.Int8{Int64: a.ActorID, Valid: a.ActorID != 0},
		Message: a.Message, Entity: a.Entity, EntityID: a.EntityID,
	})
	return toActivity(row), mapErr(err)
}

func (s *Store) ListActivity(ctx context.Context, f d.ActivityFilter) ([]d.Activity, int, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}
	q := escapeLike(f.Query)
	rows, err := s.q.ListActivity(ctx, db.ListActivityParams{
		ActorID: f.ActorID, Kind: f.Kind, ExcludeAuth: f.ExcludeAuth, NotifyMember: f.NotifyMember, Q: q,
		Lim: int32(min(limit, math.MaxInt32)), Off: int32(max(f.Offset, 0)),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountActivity(ctx, db.CountActivityParams{ActorID: f.ActorID, Kind: f.Kind, ExcludeAuth: f.ExcludeAuth, NotifyMember: f.NotifyMember, Q: q})
	if err != nil {
		return nil, 0, err
	}
	out := make([]d.Activity, len(rows))
	for i, a := range rows {
		out[i] = toActivity(a)
	}
	return out, int(total), nil
}

// timePtr maps the zero time to NULL for optional query arguments.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
