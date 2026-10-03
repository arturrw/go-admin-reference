package domain

import "time"

type RevenuePoint struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Current  int64  `json:"current"`
	Previous int64  `json:"previous"`
}

type KPI struct {
	Key      string    `json:"key"`
	Label    string    `json:"label"`
	Value    float64   `json:"value"`
	Unit     string    `json:"unit"` // count | percent | cents
	DeltaPct float64   `json:"deltaPct"`
	Trend    []float64 `json:"trend"`
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

type Activity struct {
	Kind    string    `json:"kind"` // role | publish | deploy | stock | refund
	Actor   string    `json:"actor"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
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
