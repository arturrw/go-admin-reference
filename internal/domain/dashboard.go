package domain

import "math"

type RevenuePoint struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Current  int64  `json:"current"`
	Previous int64  `json:"previous"`
}

type KPI struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Value    float64    `json:"value"`
	Unit     string     `json:"unit"` // count | percent | cents
	DeltaPct float64    `json:"deltaPct"`
	Trend    []float64  `json:"trend"`  // downsampled, for sparklines
	Series   []KPIPoint `json:"series"` // one point per day, for the detail chart
}

type KPIPoint struct {
	Date     string  `json:"date"` // YYYY-MM-DD
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
}

type CategoryShare struct {
	Category   Category `json:"category"`
	SalesCents int64    `json:"salesCents"`
}

type TopProduct struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Category     Category `json:"category"`
	Hue          int      `json:"hue"`
	ImageURL     string   `json:"imageUrl"`
	Sold         int      `json:"sold"`
	RevenueCents int64    `json:"revenueCents"`
}

type Market struct {
	Country  string  `json:"country"`
	Name     string  `json:"name"`
	SharePct float64 `json:"sharePct"`
}

type Target struct {
	Label       string  `json:"label"`
	BookedCents int64   `json:"bookedCents"`
	GoalCents   int64   `json:"goalCents"`
	PacePct     float64 `json:"pacePct"`
}

type Dashboard struct {
	RangeDays        int             `json:"rangeDays"`
	RevenueCents     int64           `json:"revenueCents"`
	PrevRevenueCents int64           `json:"prevRevenueCents"`
	Revenue          []RevenuePoint  `json:"revenue"`
	KPIs             []KPI           `json:"kpis"`
	OrdersHeatmap    [7][24]int      `json:"ordersHeatmap"` // [Mon..Sun][hour]
	Categories       []CategoryShare `json:"categories"`
	TopProducts      []TopProduct    `json:"topProducts"`
	RecentOrders     []Order         `json:"recentOrders"`
	Activity         []Activity      `json:"activity"`
	Markets          []Market        `json:"markets"`
	Target           Target          `json:"target"`
}

// Synthetic storefront metrics: there is no real storefront behind this
// reference app, so conversion and AOV are fixed and orders are derived from
// revenue. Shared by every store so dashboards look identical.
const avgOrderCents = 8640

// BuildDashboard fills the revenue-derived parts of the dashboard (totals,
// KPIs, target, markets) from a daily revenue series.
func BuildDashboard(series []RevenuePoint, markets []Market) Dashboard {
	var cur, prev int64
	for _, p := range series {
		cur += p.Current
		prev += p.Previous
	}
	orders := float64(cur) / avgOrderCents
	prevOrders := float64(prev) / avgOrderCents
	trend := bucket(series, 16)
	daily := func(f func(i int, rev float64) float64) []KPIPoint {
		out := make([]KPIPoint, len(series))
		for i, p := range series {
			out[i] = KPIPoint{Date: p.Date, Current: f(i, float64(p.Current)), Previous: f(i+len(series), float64(p.Previous))}
		}
		return out
	}
	ordersOf := func(_ int, rev float64) float64 { return math.Round(rev / avgOrderCents) }
	return Dashboard{
		RangeDays:        len(series),
		RevenueCents:     cur,
		PrevRevenueCents: prev,
		Revenue:          series,
		Markets:          markets,
		Categories:       []CategoryShare{},
		TopProducts:      []TopProduct{},
		Activity:         []Activity{},
		KPIs: []KPI{
			{Key: "orders", Label: "Orders", Value: math.Round(orders), Unit: "count", DeltaPct: pct(orders, prevOrders), Trend: trend,
				Series: daily(ordersOf)},
			{Key: "customers", Label: "New customers", Value: math.Round(orders * .62), Unit: "count", DeltaPct: pct(orders, prevOrders) * .55, Trend: wobble(trend, 1),
				Series: daily(func(i int, rev float64) float64 {
					return math.Round(ordersOf(i, rev) * .62 * (1 + .12*math.Sin(float64(i)+1)))
				})},
			{Key: "conversion", Label: "Conversion", Value: 3.84, Unit: "percent", DeltaPct: -0.6, Trend: wobble(reversed(trend), 2),
				Series: daily(func(i int, _ float64) float64 {
					return math.Round(3.84*(1+.07*math.Sin(float64(i)*.9)+.03*math.Sin(float64(i)*2.3))*100) / 100
				})},
			{Key: "aov", Label: "Avg. order value", Value: avgOrderCents, Unit: "cents", DeltaPct: 2.1, Trend: wobble(trend, 3),
				Series: daily(func(i int, _ float64) float64 {
					return math.Round(avgOrderCents * (1 + .06*math.Sin(float64(i)*1.3) + .03*math.Sin(float64(i)*.4)))
				})},
		},
		Target: Target{Label: "Q4 target", BookedCents: 34_128_000, GoalCents: 120_000_000, PacePct: 6.2},
	}
}

func pct(cur, prev float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Round((cur/prev-1)*1000) / 10
}

// bucket downsamples the revenue series to n averaged points.
func bucket(series []RevenuePoint, n int) []float64 {
	out := make([]float64, n)
	if len(series) == 0 {
		return out
	}
	for i := range n {
		lo, hi := i*len(series)/n, max((i+1)*len(series)/n, i*len(series)/n+1)
		hi = min(hi, len(series))
		var sum float64
		for _, p := range series[lo:hi] {
			sum += float64(p.Current)
		}
		out[i] = sum / float64(max(hi-lo, 1))
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

func reversed(src []float64) []float64 {
	out := make([]float64, len(src))
	for i, v := range src {
		out[len(src)-1-i] = v
	}
	return out
}
