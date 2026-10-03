package domain

import (
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
