package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arturrw/go-admin-reference/internal/seed"
)

// SeedIfEmpty loads the demo dataset into a fresh database (no members yet).
// It reports whether anything was inserted.
func SeedIfEmpty(ctx context.Context, pool *pgxpool.Pool, now time.Time) (bool, error) {
	var members int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM members").Scan(&members); err != nil {
		return false, err
	}
	if members > 0 {
		return false, nil
	}
	return true, pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return load(ctx, tx, seed.Generate(now)) })
}

// load bulk-inserts the dataset with COPY, keeping the generated ids, then
// moves every identity sequence past them.
func load(ctx context.Context, tx pgx.Tx, ds *seed.Dataset) error {
	copyRows := func(table string, cols []string, rows [][]any) error {
		_, err := tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
		if err != nil {
			return fmt.Errorf("seed %s: %w", table, err)
		}
		return nil
	}

	var rows [][]any
	for _, m := range ds.Members {
		rows = append(rows, []any{m.ID, m.Name, m.Email, string(m.Role), string(m.Status), m.MFA, m.PasswordHash, m.LastActiveAt})
	}
	if err := copyRows("members", []string{"id", "name", "email", "role", "status", "mfa", "password_hash", "last_active_at"}, rows); err != nil {
		return err
	}

	rows, imgs := nil, [][]any(nil)
	for _, p := range ds.Products {
		trend := make([]int32, len(p.Trend))
		for i, v := range p.Trend {
			trend[i] = int32(v)
		}
		rows = append(rows, []any{
			p.ID, p.Name, p.SKU, string(p.Category), p.Vendor, p.Tags, p.PriceCents, p.CompareAtCents, p.CostCents,
			int32(p.Stock), int32(p.WeightGrams), string(p.Status), int32(p.Sold30d), float32(p.Rating), int32(p.Hue),
			trend, p.Description, p.CreatedAt, p.UpdatedAt,
		})
		for i, img := range p.Images {
			imgs = append(imgs, []any{img.ID, p.ID, int32(i), img.URL, img.Alt, img.Generated, img.SizeBytes})
		}
	}
	if err := copyRows("products", []string{
		"id", "name", "sku", "category", "vendor", "tags", "price_cents", "compare_at_cents", "cost_cents",
		"stock", "weight_grams", "status", "sold_30d", "rating", "hue", "trend", "description", "created_at", "updated_at",
	}, rows); err != nil {
		return err
	}
	if err := copyRows("product_images", []string{"id", "product_id", "position", "url", "alt", "generated", "size_bytes"}, imgs); err != nil {
		return err
	}

	rows = nil
	var notes [][]any
	for _, c := range ds.Customers {
		rows = append(rows, []any{
			c.ID, c.Name, c.Email, c.Phone, c.Country, c.Address.Line1, c.Address.City, c.Address.PostalCode,
			c.Tags, c.AcceptsMarketing, c.Source, c.LastSeenAt, c.CreatedAt,
		})
		for _, n := range c.Notes {
			notes = append(notes, []any{n.ID, c.ID, n.Author, n.Text, n.At})
		}
	}
	if err := copyRows("customers", []string{
		"id", "name", "email", "phone", "country", "address_line1", "city", "postal_code",
		"tags", "accepts_marketing", "source", "last_seen_at", "created_at",
	}, rows); err != nil {
		return err
	}
	if err := copyRows("customer_notes", []string{"id", "customer_id", "author", "body", "created_at"}, notes); err != nil {
		return err
	}

	rows = nil
	var items [][]any
	for _, o := range ds.Orders {
		rows = append(rows, []any{o.ID, o.Customer.ID, string(o.Status), o.Payment, o.TotalCents, o.PlacedAt})
		for i, it := range o.Items {
			items = append(items, []any{o.ID, int32(i), it.ProductID, it.Name, it.SKU, string(it.Category), int32(it.Hue), it.ImageURL, int32(it.Qty), it.PriceCents})
		}
	}
	if err := copyRows("orders", []string{"id", "customer_id", "status", "payment", "total_cents", "placed_at"}, rows); err != nil {
		return err
	}
	if err := copyRows("order_items", []string{"order_id", "line", "product_id", "name", "sku", "category", "hue", "image_url", "qty", "price_cents"}, items); err != nil {
		return err
	}

	rows = nil
	for _, r := range ds.Revenue {
		day, err := time.Parse(time.DateOnly, r.Date)
		if err != nil {
			return err
		}
		rows = append(rows, []any{day, r.Current, r.Previous})
	}
	if err := copyRows("revenue_daily", []string{"day", "current_cents", "previous_cents"}, rows); err != nil {
		return err
	}

	rows = nil
	for dow, hours := range ds.Heatmap {
		for h, n := range hours {
			rows = append(rows, []any{int16(dow), int16(h), int32(n)})
		}
	}
	if err := copyRows("orders_heatmap", []string{"dow", "hour", "orders"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, a := range ds.Activity {
		var actorID *int64
		if a.ActorID != 0 {
			actorID = &a.ActorID
		}
		rows = append(rows, []any{a.ID, a.Kind, a.Actor, actorID, a.Message, a.Entity, a.EntityID, a.At})
	}
	if err := copyRows("activity", []string{"id", "kind", "actor", "actor_id", "message", "entity", "entity_id", "at"}, rows); err != nil {
		return err
	}

	// Explicit ids were inserted, so advance the identity sequences past them.
	for _, t := range []string{"members", "products", "customers", "customer_notes", "orders", "activity"} {
		q := fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%[1]s', 'id'), coalesce((SELECT max(id) FROM %[1]s), 0) + 1, false)`, t)
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("reset %s sequence: %w", t, err)
		}
	}
	return nil
}

// UseSeedPhotos swaps the generated SVG artwork in databases seeded before
// stock photos existed. Only untouched seed slots are updated, so images a
// user uploaded or re-ordered stay as they are. It reports how many changed.
func UseSeedPhotos(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	b := &pgx.Batch{}
	for _, img := range seed.SeedImages() {
		b.Queue(`UPDATE product_images pi SET url = $1, alt = $2
			FROM products p
			WHERE pi.product_id = p.id AND p.name = $3
			  AND pi.id = 'gen-' || p.id || '-' || $4::int
			  AND pi.url LIKE '/media/generated/%'`, img.URL, img.Alt, img.ProductName, img.Position)
	}
	res := pool.SendBatch(ctx, b)
	defer res.Close()
	var n int64
	for range b.Len() {
		tag, err := res.Exec()
		if err != nil {
			return n, err
		}
		n += tag.RowsAffected()
	}
	return n, nil
}
