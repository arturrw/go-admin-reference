package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// Dashboard loads the orders of the period (and of the quarter, for the
// target) and builds the figures with the same code the in-memory store uses.
// Reading them as rows rather than aggregating in SQL is deliberate: one place
// owns the rules (which orders count, what a day is), so the two stores cannot
// drift apart.
func (s *Store) Dashboard(ctx context.Context, days int) (d.Dashboard, error) {
	now := s.now()
	days = d.DashboardDays(days)
	since := d.SalesSince(now, days)
	in := d.DashboardInput{Now: now, Days: days}

	sales, err := s.loadSales(ctx, since, nil)
	if err != nil {
		return d.Dashboard{}, err
	}
	in.Sales = sales

	rows, err := s.pool.Query(ctx, `SELECT created_at FROM customers WHERE created_at >= $1`, since)
	if err != nil {
		return d.Dashboard{}, err
	}
	in.CustomersSince, err = pgx.CollectRows(rows, pgx.RowTo[time.Time])
	if err != nil {
		return d.Dashboard{}, err
	}

	key, _, _ := d.QuarterOf(now)
	if in.Goal, err = s.TargetGoal(ctx, key); err != nil {
		return d.Dashboard{}, err
	}
	dash := d.BuildDashboard(in)

	if dash.RecentOrders, _, err = s.ListOrders(ctx, d.OrderFilter{Limit: 6}); err != nil {
		return d.Dashboard{}, err
	}
	if dash.Activity, _, err = s.ListActivity(ctx, d.ActivityFilter{ExcludeAuth: true, Limit: d.DashboardActivity}); err != nil {
		return d.Dashboard{}, err
	}
	return dash, nil
}

// loadSales reads the orders placed since a time, with their items. With
// product ids it keeps only the orders that contain one of them (and only
// those items), which is all a product's own sales need.
func (s *Store) loadSales(ctx context.Context, since time.Time, productIDs []int64) ([]d.Sale, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.status, o.placed_at, o.total_cents, c.country
		FROM orders o JOIN customers c ON c.id = o.customer_id
		WHERE o.placed_at >= $1
		  AND ($2::bigint[] IS NULL OR EXISTS (
		        SELECT 1 FROM order_items i WHERE i.order_id = o.id AND i.product_id = ANY($2)))`, since, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sales []d.Sale
	index := map[int64]int{}
	for rows.Next() {
		var id int64
		var sale d.Sale
		var status string
		if err := rows.Scan(&id, &status, &sale.PlacedAt, &sale.TotalCents, &sale.Country); err != nil {
			return nil, err
		}
		sale.Status = d.OrderStatus(status)
		index[id] = len(sales)
		sales = append(sales, sale)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}

	items, err := s.pool.Query(ctx, `
		SELECT order_id, coalesce(product_id, 0), name, category, hue, image_url, qty, price_cents
		FROM order_items WHERE order_id = ANY($1) AND ($2::bigint[] IS NULL OR product_id = ANY($2))
		ORDER BY order_id, line`, ids, productIDs)
	if err != nil {
		return nil, err
	}
	defer items.Close()
	for items.Next() {
		var orderID int64
		var it d.OrderItem
		var category string
		if err := items.Scan(&orderID, &it.ProductID, &it.Name, &category, &it.Hue, &it.ImageURL, &it.Qty, &it.PriceCents); err != nil {
			return nil, err
		}
		it.Category = d.Category(category)
		sale := &sales[index[orderID]]
		sale.Items = append(sale.Items, it)
	}
	return sales, items.Err()
}
