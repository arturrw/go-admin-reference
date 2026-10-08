package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	d "github.com/arturrw/go-admin-reference/internal/domain"
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
		rows = append(rows, []any{m.ID, m.Name, m.Email, string(m.Role), string(m.Status), m.MFA, m.PasswordHash, m.LastActiveAt, fromPerms(m.Granted), fromPerms(m.Revoked)})
	}
	if err := copyRows("members", []string{"id", "name", "email", "role", "status", "mfa", "password_hash", "last_active_at", "granted", "revoked"}, rows); err != nil {
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
		var reason, by string
		var at *time.Time
		if o.Refund != nil {
			reason, by, at = o.Refund.Reason, o.Refund.By, &o.Refund.At
		}
		rows = append(rows, []any{o.ID, o.Customer.ID, string(o.Status), o.Payment, o.TotalCents, o.PlacedAt, reason, by, at})
		for i, it := range o.Items {
			items = append(items, []any{o.ID, int32(i), it.ProductID, it.Name, it.SKU, string(it.Category), int32(it.Hue), it.ImageURL, int32(it.Qty), it.PriceCents})
		}
	}
	if err := copyRows("orders", []string{"id", "customer_id", "status", "payment", "total_cents", "placed_at", "refund_reason", "refunded_by", "refunded_at"}, rows); err != nil {
		return err
	}
	if err := copyRows("order_items", []string{"order_id", "line", "product_id", "name", "sku", "category", "hue", "image_url", "qty", "price_cents"}, items); err != nil {
		return err
	}

	rows = nil
	for _, o := range ds.Orders {
		for i, e := range o.Events {
			rows = append(rows, []any{o.ID, int32(i + 1), string(e.Status), e.At, e.By})
		}
	}
	if err := copyRows("order_events", []string{"order_id", "seq", "status", "at", "by"}, rows); err != nil {
		return err
	}

	rows = nil
	for _, k := range ds.APIKeys {
		rows = append(rows, []any{k.Name, k.Scope, k.Last4, k.Hash, k.CreatedBy, k.CreatedAt, k.LastUsedAt})
	}
	if err := copyRows("api_keys", []string{"name", "scope", "last4", "token_hash", "created_by", "created_at", "last_used_at"}, rows); err != nil {
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

// BackfillRefundReasons gives refunded orders in databases seeded before
// refunds kept a reason the same kind of demo reason a fresh seed has. Orders
// refunded through the API already have one and are left alone. It reports
// how many changed.
func BackfillRefundReasons(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE orders
		SET refund_reason = ($1::text[])[id % cardinality($1::text[]) + 1],
		    refunded_by   = ($2::text[])[id % cardinality($2::text[]) + 1],
		    refunded_at   = least(placed_at + interval '2 days', now())
		WHERE status = 'refunded' AND refunded_at IS NULL`,
		d.RefundReasons, []string{"Priya Shah", "Diego Vega"})
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// legacyActivity are the feed rows the first schema seeded, with no actor or
// record attached. The audit-log seed replaces them.
var legacyActivity = []string{
	"changed Sofia Rossi's role to editor", "published Pulse Watch Ultra", "rolled out v1.4.2 to 3/3 pods",
	"Glow Strip 5m dropped below 10 units", "issued a $129.99 refund",
}

// BackfillActivity gives databases seeded before the audit log the same week
// of staff history a fresh seed has, so team members have activity to show.
// Refund entries use the reasons stored on the orders, entries of members
// that no longer exist lose their actor link, and a database that already has
// the history is left alone. It reports how many entries were added.
func BackfillActivity(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
	var done bool
	if err := pool.QueryRow(ctx, `SELECT exists(SELECT 1 FROM activity WHERE kind = 'deploy' AND message = 'rolled out v1.4.1 to 3/3 pods')`).Scan(&done); err != nil || done {
		return 0, err
	}

	ds := seed.Generate(now)
	type refund struct {
		reason, by string
		at         time.Time
	}
	refunds := map[int64]refund{}
	rows, err := pool.Query(ctx, `SELECT id, refund_reason, refunded_by, refunded_at FROM orders WHERE refunded_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var id int64
		var r refund
		if err := rows.Scan(&id, &r.reason, &r.by, &r.at); err != nil {
			return 0, err
		}
		refunds[id] = r
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for i := range ds.Orders {
		o := &ds.Orders[i]
		o.Refund = nil
		if r, ok := refunds[o.ID]; ok {
			o.Refund = &d.OrderRefund{Reason: r.reason, By: r.by, At: r.at}
		}
	}

	members := map[int64]bool{}
	if rows, err = pool.Query(ctx, `SELECT id FROM members`); err != nil {
		return 0, err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		members[id] = true
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	acts := seed.BuildActivity(ds, now)
	return len(acts), pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM activity WHERE actor_id IS NULL AND entity = '' AND message = ANY($1)`, legacyActivity); err != nil {
			return err
		}
		var copyRows [][]any
		for _, a := range acts {
			var actorID *int64
			if a.ActorID != 0 && members[a.ActorID] {
				actorID = &a.ActorID
			}
			copyRows = append(copyRows, []any{a.Kind, a.Actor, actorID, a.Message, a.Entity, a.EntityID, a.At})
		}
		_, err := tx.CopyFrom(ctx, pgx.Identifier{"activity"}, []string{"kind", "actor", "actor_id", "message", "entity", "entity_id", "at"}, pgx.CopyFromRows(copyRows))
		return err
	})
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
