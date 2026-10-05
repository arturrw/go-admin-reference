package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPaid      OrderStatus = "paid"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
	OrderRefunded  OrderStatus = "refunded"
	OrderFailed    OrderStatus = "failed"
)

var OrderStatuses = []OrderStatus{OrderPending, OrderPaid, OrderShipped, OrderDelivered, OrderRefunded, OrderFailed}

// Counts toward revenue / lifetime value.
func (s OrderStatus) Billable() bool { return s != OrderRefunded && s != OrderFailed }

// OrderItem is a snapshot of the product at purchase time.
type OrderItem struct {
	ProductID  int64    `json:"productId"`
	Name       string   `json:"name"`
	SKU        string   `json:"sku"`
	Category   Category `json:"category"`
	Hue        int      `json:"hue"`
	ImageURL   string   `json:"imageUrl"`
	Qty        int      `json:"qty"`
	PriceCents int64    `json:"priceCents"`
}

type CustomerRef struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Country string `json:"country"`
	Segment string `json:"segment"`
}

type Order struct {
	ID         int64       `json:"id"`
	Customer   CustomerRef `json:"customer"`
	Items      []OrderItem `json:"items"`
	TotalCents int64       `json:"totalCents"`
	Status     OrderStatus `json:"status"`
	Payment    string      `json:"payment"`
	PlacedAt   time.Time   `json:"placedAt"`
}

type OrderFilter struct {
	Query      string
	Status     OrderStatus
	CustomerID int64
	From, To   time.Time // placed_at in [From, To); zero = unbounded
	Limit      int
	Offset     int
}

// ── Customers ───────────────────────────────────────────────────────────────

type Address struct {
	Line1      string `json:"line1"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type CustomerNote struct {
	ID     int64     `json:"id"`
	Author string    `json:"author"`
	Text   string    `json:"text"`
	At     time.Time `json:"at"`
}

type Customer struct {
	ID               int64          `json:"id"`
	Name             string         `json:"name"`
	Email            string         `json:"email"`
	Phone            string         `json:"phone"`
	Country          string         `json:"country"`
	Address          Address        `json:"address"`
	Orders           int            `json:"orders"`
	LTVCents         int64          `json:"ltvCents"`
	Segment          string         `json:"segment"` // VIP | Regular | New | At risk
	Tags             []string       `json:"tags"`
	AcceptsMarketing bool           `json:"acceptsMarketing"`
	Source           string         `json:"source"`
	Notes            []CustomerNote `json:"notes"`
	LastSeenAt       time.Time      `json:"lastSeenAt"`
	LastOrderAt      time.Time      `json:"lastOrderAt"`
	CreatedAt        time.Time      `json:"createdAt"`
}

func (c Customer) Ref() CustomerRef {
	return CustomerRef{ID: c.ID, Name: c.Name, Email: c.Email, Country: c.Country, Segment: c.Segment}
}

type CustomerFilter struct {
	Query   string
	Segment string
}

type SegmentSummary struct {
	Count    int   `json:"count"`
	LTVCents int64 `json:"ltvCents"`
}

type CustomerStats struct {
	TotalSpentCents int64     `json:"totalSpentCents"`
	Orders          int       `json:"orders"`
	AvgOrderCents   int64     `json:"avgOrderCents"`
	ItemsBought     int       `json:"itemsBought"`
	Refunds         int       `json:"refunds"`
	FirstOrderAt    time.Time `json:"firstOrderAt"`
	LastOrderAt     time.Time `json:"lastOrderAt"`
}

type PurchasedProduct struct {
	ProductID  int64    `json:"productId"`
	Name       string   `json:"name"`
	Category   Category `json:"category"`
	Hue        int      `json:"hue"`
	ImageURL   string   `json:"imageUrl"`
	Qty        int      `json:"qty"`
	SpentCents int64    `json:"spentCents"`
}

type MonthlySpend struct {
	Month string `json:"month"` // YYYY-MM
	Cents int64  `json:"cents"`
}

// CustomerDetail is everything the customer drawer shows.
type CustomerDetail struct {
	Customer   Customer           `json:"customer"`
	Stats      CustomerStats      `json:"stats"`
	Orders     []Order            `json:"orders"`
	Products   []PurchasedProduct `json:"products"`
	Categories []CategoryShare    `json:"categories"`
	Monthly    []MonthlySpend     `json:"monthly"`
}

func ValidateNote(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		return "", NewValidationError("text", "is required")
	case len(text) > 2000:
		return "", NewValidationError("text", "must be at most 2000 characters")
	}
	return text, nil
}

// Segmentation thresholds. The Postgres view customers_v mirrors these rules.
const (
	AtRiskAfter     = 45 * 24 * time.Hour
	NewWithin       = 30 * 24 * time.Hour
	VIPLifetimeCent = 150_000
	VIPOrders       = 8
)

// CustomerSegment classifies a customer from their billable order history.
func CustomerSegment(orders int, ltvCents int64, lastOrderAt, createdAt, now time.Time) string {
	switch {
	case !lastOrderAt.IsZero() && now.Sub(lastOrderAt) > AtRiskAfter:
		return "At risk"
	case ltvCents >= VIPLifetimeCent || orders >= VIPOrders:
		return "VIP"
	case orders <= 1 || (!createdAt.IsZero() && now.Sub(createdAt) < NewWithin):
		return "New"
	default:
		return "Regular"
	}
}

// DeriveCustomer recomputes order count, lifetime value, last order and
// segment from the given orders.
func DeriveCustomer(c *Customer, orders []Order, now time.Time) {
	c.Orders, c.LTVCents, c.LastOrderAt = 0, 0, time.Time{}
	for _, o := range orders {
		if o.Customer.ID != c.ID {
			continue
		}
		if o.PlacedAt.After(c.LastOrderAt) {
			c.LastOrderAt = o.PlacedAt
		}
		if o.Status.Billable() {
			c.Orders++
			c.LTVCents += o.TotalCents
		}
	}
	c.Segment = CustomerSegment(c.Orders, c.LTVCents, c.LastOrderAt, c.CreatedAt, now)
}

// BuildCustomerDetail aggregates a customer's orders (newest first) into the
// detail view: totals, purchased products, category split and monthly spend.
func BuildCustomerDetail(c Customer, orders []Order, now time.Time) CustomerDetail {
	slices.SortFunc(c.Notes, func(a, b CustomerNote) int { return b.At.Compare(a.At) })
	det := CustomerDetail{Customer: c, Orders: orders}
	if det.Orders == nil {
		det.Orders = []Order{}
	}
	products := map[int64]*PurchasedProduct{}
	categories := map[Category]int64{}
	monthly := map[string]int64{}
	for _, o := range orders {
		if o.Status == OrderRefunded {
			det.Stats.Refunds++
		}
		if !o.Status.Billable() {
			continue
		}
		det.Stats.TotalSpentCents += o.TotalCents
		det.Stats.Orders++
		monthly[o.PlacedAt.Format("2006-01")] += o.TotalCents
		for _, it := range o.Items {
			det.Stats.ItemsBought += it.Qty
			categories[it.Category] += it.PriceCents * int64(it.Qty)
			pp, ok := products[it.ProductID]
			if !ok {
				pp = &PurchasedProduct{ProductID: it.ProductID, Name: it.Name, Category: it.Category, Hue: it.Hue, ImageURL: it.ImageURL}
				products[it.ProductID] = pp
			}
			pp.Qty += it.Qty
			pp.SpentCents += it.PriceCents * int64(it.Qty)
		}
	}
	if n := len(orders); n > 0 {
		det.Stats.LastOrderAt = orders[0].PlacedAt
		det.Stats.FirstOrderAt = orders[n-1].PlacedAt
	}
	if det.Stats.Orders > 0 {
		det.Stats.AvgOrderCents = det.Stats.TotalSpentCents / int64(det.Stats.Orders)
	}

	det.Products = make([]PurchasedProduct, 0, len(products))
	for _, p := range products {
		det.Products = append(det.Products, *p)
	}
	slices.SortFunc(det.Products, func(a, b PurchasedProduct) int {
		return cmp.Or(cmp.Compare(b.SpentCents, a.SpentCents), cmp.Compare(a.ProductID, b.ProductID))
	})

	det.Categories = []CategoryShare{}
	for _, cat := range Categories {
		if v := categories[cat]; v > 0 {
			det.Categories = append(det.Categories, CategoryShare{Category: cat, SalesCents: v})
		}
	}
	slices.SortFunc(det.Categories, func(a, b CategoryShare) int { return cmp.Compare(b.SalesCents, a.SalesCents) })

	// Last 6 calendar months, oldest first, zero-filled.
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for k := 5; k >= 0; k-- {
		m := first.AddDate(0, -k, 0).Format("2006-01")
		det.Monthly = append(det.Monthly, MonthlySpend{Month: m, Cents: monthly[m]})
	}
	return det
}
