package domain

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// Sale is the part of an order the dashboard needs. Both stores load these
// and hand them to BuildDashboard, so memory and Postgres show the same numbers.
type Sale struct {
	PlacedAt   time.Time
	Status     OrderStatus
	TotalCents int64
	Country    string
	Items      []OrderItem
}

// DashboardInput is everything BuildDashboard works from.
type DashboardInput struct {
	Now  time.Time
	Days int
	// Sales are all orders placed since SalesSince(Now, Days).
	Sales []Sale
	// CustomersSince are the sign-up times of customers created since SalesSince.
	CustomersSince []time.Time
	Goal           *TargetGoal // the owner's goal for this quarter, nil while unset
}

// DashboardDays clamps a requested range to what the dashboard offers.
func DashboardDays(days int) int { return min(max(days, 7), 90) }

// SalesSince is how far back the stores must load orders: the previous period
// (for the deltas) and the whole current quarter (for the target).
func SalesSince(now time.Time, days int) time.Time {
	_, qStart, _ := QuarterOf(now.UTC())
	return minTime(utcDay(now).AddDate(0, 0, -(2*days-1)), qStart)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func utcDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Conversion needs storefront traffic this app does not have; it stays a fixed
// sample, flagged as such.
const sampleConversion = 3.84

var countryNames = map[string]string{
	"US": "United States", "DE": "Germany", "GB": "United Kingdom", "FR": "France", "NL": "Netherlands",
	"PL": "Poland", "CA": "Canada", "JP": "Japan", "SE": "Sweden", "BR": "Brazil", "ES": "Spain",
}

// BuildDashboard computes revenue, KPIs, the heatmap, category and product
// sales, markets and the quarter target from orders. Days are UTC calendar days
// ending today; the previous period is the same number of days before them.
// Revenue and sales count orders that are not refunded or failed (the rule the
// customers' lifetime value uses); the order count and heatmap count every
// order that was not a failed payment.
func BuildDashboard(in DashboardInput) Dashboard {
	days := DashboardDays(in.Days)
	today := utcDay(in.Now)
	curStart := today.AddDate(0, 0, -(days - 1))
	prevStart := curStart.AddDate(0, 0, -days)
	dayIndex := func(t time.Time) int {
		if t.Before(prevStart) {
			return -1
		}
		return int(utcDay(t).Sub(prevStart) / (24 * time.Hour))
	}

	revenue := make([]int64, 2*days) // previous period first, then current
	orders := make([]float64, 2*days)
	billable := make([]float64, 2*days)
	newCust := make([]float64, 2*days)
	var heat [7][24]int
	catSales := map[Category]int64{}
	type tally struct {
		item    OrderItem
		sold    int
		revenue int64
	}
	byProduct := map[int64]*tally{}
	byCountry := map[string]int{}
	var countryOrders int
	var booked int64
	_, qStart, _ := QuarterOf(in.Now.UTC())

	for _, s := range in.Sales {
		if s.Status.Billable() && !s.PlacedAt.Before(qStart) {
			booked += s.TotalCents
		}
		i := dayIndex(s.PlacedAt)
		if i < 0 || i >= 2*days {
			continue
		}
		if s.Status != OrderFailed {
			orders[i]++
		}
		if s.Status.Billable() {
			revenue[i] += s.TotalCents
			billable[i]++
		}
		if i < days {
			continue
		}
		// The current period only.
		if s.Status != OrderFailed {
			t := s.PlacedAt.UTC()
			heat[(int(t.Weekday())+6)%7][t.Hour()]++
			byCountry[s.Country]++
			countryOrders++
		}
		if !s.Status.Billable() {
			continue
		}
		for _, it := range s.Items {
			line := it.PriceCents * int64(it.Qty)
			catSales[it.Category] += line
			if it.ProductID == 0 {
				continue
			}
			t := byProduct[it.ProductID]
			if t == nil {
				t = &tally{item: it}
				byProduct[it.ProductID] = t
			}
			t.sold += it.Qty
			t.revenue += line
		}
	}
	for _, c := range in.CustomersSince {
		if i := dayIndex(c); i >= 0 && i < 2*days {
			newCust[i]++
		}
	}

	series := make([]RevenuePoint, days)
	var cur, prev int64
	for i := range days {
		series[i] = RevenuePoint{Date: curStart.AddDate(0, 0, i).Format(time.DateOnly), Current: revenue[days+i], Previous: revenue[i]}
		cur += revenue[days+i]
		prev += revenue[i]
	}

	sum := func(v []float64, from, to int) (t float64) {
		for _, x := range v[from:to] {
			t += x
		}
		return t
	}
	// daily builds a KPI's per-day series; f gets the day's index into the
	// 2*days arrays.
	daily := func(f func(i int) float64) []KPIPoint {
		out := make([]KPIPoint, days)
		for i := range days {
			out[i] = KPIPoint{Date: series[i].Date, Current: f(days + i), Previous: f(i)}
		}
		return out
	}
	kpi := func(key, label, unit string, value, previous float64, f func(i int) float64) KPI {
		pts := daily(f)
		vals := make([]float64, len(pts))
		for i, p := range pts {
			vals[i] = p.Current
		}
		return KPI{Key: key, Label: label, Unit: unit, Value: value, DeltaPct: pct(value, previous), Trend: bucket(vals, 16), Series: pts}
	}
	aov := func(rev int64, n float64) float64 {
		if n == 0 {
			return 0
		}
		return math.Round(float64(rev) / n)
	}
	conv := kpi("conversion", "Conversion", "percent", sampleConversion, sampleConversion, func(i int) float64 {
		return math.Round(sampleConversion*(1+.07*math.Sin(float64(i)*.9)+.03*math.Sin(float64(i)*2.3))*100) / 100
	})
	conv.DeltaPct, conv.Synthetic = -0.6, true

	dash := Dashboard{
		RangeDays:        days,
		RevenueCents:     cur,
		PrevRevenueCents: prev,
		Revenue:          series,
		OrdersHeatmap:    heat,
		Activity:         []Activity{},
		RecentOrders:     []Order{},
		Categories:       []CategoryShare{},
		TopProducts:      []TopProduct{},
		Markets:          []Market{},
		KPIs: []KPI{
			kpi("orders", "Orders", "count", sum(orders, days, 2*days), sum(orders, 0, days), func(i int) float64 { return orders[i] }),
			kpi("customers", "New customers", "count", sum(newCust, days, 2*days), sum(newCust, 0, days), func(i int) float64 { return newCust[i] }),
			conv,
			kpi("aov", "Avg. order value", "cents", aov(cur, sum(billable, days, 2*days)), aov(prev, sum(billable, 0, days)),
				func(i int) float64 { return aov(revenue[i], billable[i]) }),
		},
		Target: BuildTarget(in.Now, in.Goal, booked),
	}

	for _, c := range Categories {
		dash.Categories = append(dash.Categories, CategoryShare{Category: c, SalesCents: catSales[c]})
	}
	slices.SortStableFunc(dash.Categories, func(a, b CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	var top []*tally
	for _, t := range byProduct {
		top = append(top, t)
	}
	slices.SortFunc(top, func(a, b *tally) int {
		return cmp.Or(cmp.Compare(b.revenue, a.revenue), cmp.Compare(a.item.ProductID, b.item.ProductID))
	})
	for _, t := range top[:min(5, len(top))] {
		dash.TopProducts = append(dash.TopProducts, TopProduct{
			ID: t.item.ProductID, Name: t.item.Name, Category: t.item.Category, Hue: t.item.Hue, ImageURL: t.item.ImageURL,
			Sold: t.sold, RevenueCents: t.revenue,
		})
	}

	type share struct {
		code string
		n    int
	}
	var shares []share
	for code, n := range byCountry {
		shares = append(shares, share{code, n})
	}
	slices.SortFunc(shares, func(a, b share) int { return cmp.Or(cmp.Compare(b.n, a.n), cmp.Compare(a.code, b.code)) })
	for _, s := range shares[:min(7, len(shares))] {
		name := countryNames[s.code]
		if name == "" {
			name = s.code
		}
		dash.Markets = append(dash.Markets, Market{Country: s.code, Name: name, SharePct: math.Round(float64(s.n)/float64(countryOrders)*1000) / 10})
	}
	return dash
}

func pct(cur, prev float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Round((cur/prev-1)*1000) / 10
}

// bucket downsamples a daily series to n averaged points for a sparkline.
func bucket(vals []float64, n int) []float64 {
	out := make([]float64, n)
	if len(vals) == 0 {
		return out
	}
	for i := range n {
		lo, hi := i*len(vals)/n, max((i+1)*len(vals)/n, i*len(vals)/n+1)
		hi = min(hi, len(vals))
		var sum float64
		for _, v := range vals[lo:hi] {
			sum += v
		}
		out[i] = sum / float64(max(hi-lo, 1))
	}
	return out
}
