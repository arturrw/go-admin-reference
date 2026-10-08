package domain

import (
	"fmt"
	"math"
	"time"
)

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
	// Synthetic marks a figure the orders cannot tell (there is no storefront
	// traffic behind this app), so the UI says it is a sample.
	Synthetic bool `json:"synthetic,omitempty"`
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

// Target is the revenue goal for the current quarter, which the owner sets.
type Target struct {
	Label       string     `json:"label"`   // "Q4 target"
	Quarter     string     `json:"quarter"` // "2026-Q4"
	Period      string     `json:"period"`  // "Oct – Dec"
	BookedCents int64      `json:"bookedCents"`
	GoalCents   int64      `json:"goalCents"`
	PacePct     float64    `json:"pacePct"` // booked share of the goal minus share of the quarter elapsed
	UpdatedBy   string     `json:"updatedBy"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}

// TargetGoal is a stored quarterly goal.
type TargetGoal struct {
	Quarter   string
	GoalCents int64
	UpdatedBy string
	UpdatedAt time.Time
}

const (
	// DefaultGoalCents applies until the owner sets a goal for the quarter.
	DefaultGoalCents = 12_000_000
	MaxGoalCents     = 10_000_000_000_000 // $100bn
)

// QuarterOf returns the quarter key ("2026-Q4") and its bounds.
func QuarterOf(t time.Time) (key string, start, end time.Time) {
	q := (int(t.Month()) - 1) / 3
	start = time.Date(t.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, t.Location())
	return fmt.Sprintf("%d-Q%d", t.Year(), q+1), start, start.AddDate(0, 3, 0)
}

// BuildTarget computes the current quarter's target; goal is nil while unset.
func BuildTarget(now time.Time, goal *TargetGoal, bookedCents int64) Target {
	key, start, end := QuarterOf(now)
	t := Target{
		Label: key[5:] + " target", Quarter: key, BookedCents: bookedCents, GoalCents: DefaultGoalCents,
		Period: start.Format("Jan") + " – " + end.AddDate(0, 0, -1).Format("Jan"),
	}
	if goal != nil {
		t.GoalCents, t.UpdatedBy = goal.GoalCents, goal.UpdatedBy
		at := goal.UpdatedAt
		t.UpdatedAt = &at
	}
	elapsed := float64(now.Sub(start)) / float64(end.Sub(start))
	t.PacePct = math.Round((float64(t.BookedCents)/float64(t.GoalCents)-elapsed)*1000) / 10
	return t
}

func ValidateGoal(cents int64) error {
	if cents <= 0 || cents > MaxGoalCents {
		return NewValidationError("goalCents", "must be between $0.01 and $100bn")
	}
	return nil
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
