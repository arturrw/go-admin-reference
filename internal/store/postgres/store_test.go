package postgres_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/store/memory"
	"github.com/arturrw/go-admin-reference/internal/store/postgres"
)

// TestParityWithMemory loads the same seed into both stores and checks that
// reads agree — in particular that the customers_v SQL view segments exactly
// like domain.CustomerSegment. Requires TEST_DATABASE_URL.
func TestParityWithMemory(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := postgres.SeedIfEmpty(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	pg, mem := postgres.New(pool), memory.New(now)

	check := func(name string, a, b any) {
		t.Helper()
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s differs:\n postgres: %+v\n memory:   %+v", name, a, b)
		}
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	ps, err := pg.ProductStats(ctx)
	must(err)
	ms, err := mem.ProductStats(ctx)
	must(err)
	check("product stats", ps, ms)

	for _, f := range []d.ProductFilter{{}, {Sort: "price"}, {Category: d.CategoryAudio, Sort: "name"}, {Query: "pro"}, {Status: d.ProductDraft}} {
		pl, err := pg.ListProducts(ctx, f)
		must(err)
		ml, err := mem.ListProducts(ctx, f)
		must(err)
		check("product ids for "+f.Sort+string(f.Category)+f.Query, ids(pl, func(p d.Product) int64 { return p.ID }), ids(ml, func(p d.Product) int64 { return p.ID }))
	}

	pc, err := pg.OrderCounts(ctx)
	must(err)
	mc, err := mem.OrderCounts(ctx)
	must(err)
	check("order counts", pc, mc)

	pseg, err := pg.CustomerSegments(ctx)
	must(err)
	mseg, err := mem.CustomerSegments(ctx)
	must(err)
	check("customer segments", pseg, mseg)

	po, ptotal, err := pg.ListOrders(ctx, d.OrderFilter{Status: d.OrderDelivered, Limit: 10, Offset: 5})
	must(err)
	mo, mtotal, err := mem.ListOrders(ctx, d.OrderFilter{Status: d.OrderDelivered, Limit: 10, Offset: 5})
	must(err)
	check("order page total", ptotal, mtotal)
	check("order page ids", ids(po, func(o d.Order) int64 { return o.ID }), ids(mo, func(o d.Order) int64 { return o.ID }))

	pd, err := pg.GetCustomer(ctx, 18)
	must(err)
	md, err := mem.GetCustomer(ctx, 18)
	must(err)
	check("customer stats", pd.Stats.TotalSpentCents, md.Stats.TotalSpentCents)
	check("customer segment", pd.Customer.Segment, md.Customer.Segment)
	check("customer order count", len(pd.Orders), len(md.Orders))

	pdash, err := pg.Dashboard(ctx, 30)
	must(err)
	mdash, err := mem.Dashboard(ctx, 30)
	must(err)
	check("dashboard revenue", pdash.RevenueCents, mdash.RevenueCents)
	check("dashboard heatmap", pdash.OrdersHeatmap, mdash.OrdersHeatmap)
	check("dashboard top products", ids(pdash.TopProducts, func(p d.TopProduct) int64 { return p.ID }), ids(mdash.TopProducts, func(p d.TopProduct) int64 { return p.ID }))
}

func ids[T any](xs []T, id func(T) int64) []int64 {
	out := make([]int64, len(xs))
	for i, x := range xs {
		out[i] = id(x)
	}
	return out
}

// Databases seeded before the audit log get the week of history once.
func TestBackfillActivity(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := postgres.SeedIfEmpty(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	if n, err := postgres.BackfillActivity(ctx, pool, now); err != nil || n != 0 {
		t.Fatalf("fresh seed: backfilled %d (%v), want 0", n, err)
	}

	// Simulate the first schema's feed: five rows with no actor or record.
	if _, err := pool.Exec(ctx, `DELETE FROM activity;
		INSERT INTO activity (kind, actor, message) VALUES ('refund', 'Priya Shah', 'issued a $129.99 refund'), ('deploy', 'CI', 'rolled out v1.4.2 to 3/3 pods')`); err != nil {
		t.Fatal(err)
	}
	n, err := postgres.BackfillActivity(ctx, pool, now)
	if err != nil || n == 0 {
		t.Fatalf("backfill: %d %v", n, err)
	}
	var total, mark, legacy int
	pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE actor_id = 2), count(*) FILTER (WHERE message = 'issued a $129.99 refund') FROM activity`).Scan(&total, &mark, &legacy)
	if total != n || mark == 0 || legacy != 0 {
		t.Fatalf("after backfill: %d rows (want %d), %d by Mark, %d legacy", total, n, mark, legacy)
	}
	if again, err := postgres.BackfillActivity(ctx, pool, now); err != nil || again != 0 {
		t.Fatalf("second run added %d (%v)", again, err)
	}
}
