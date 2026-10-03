// Package domain holds the core types shared by storage and transport layers.
// Money is always stored as integer cents.
package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// ValidationError maps field names to human-readable problems.
type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	slices.Sort(parts)
	return "validation failed: " + strings.Join(parts, "; ")
}

type validator map[string]string

func (v validator) check(ok bool, field, msg string) {
	if !ok {
		if _, exists := v[field]; !exists {
			v[field] = msg
		}
	}
}

func (v validator) err() error {
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Fields: v}
}

// ── Products ────────────────────────────────────────────────────────────────

type Category string

const (
	CategoryAudio       Category = "Audio"
	CategoryWearables   Category = "Wearables"
	CategoryLighting    Category = "Lighting"
	CategoryHome        Category = "Home"
	CategoryComputing   Category = "Computing"
	CategoryAccessories Category = "Accessories"
)

var Categories = []Category{CategoryAudio, CategoryWearables, CategoryLighting, CategoryHome, CategoryComputing, CategoryAccessories}

type ProductStatus string

const (
	ProductActive   ProductStatus = "active"
	ProductDraft    ProductStatus = "draft"
	ProductArchived ProductStatus = "archived"
)

type Product struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	SKU         string        `json:"sku"`
	Category    Category      `json:"category"`
	PriceCents  int64         `json:"priceCents"`
	Stock       int           `json:"stock"`
	Status      ProductStatus `json:"status"`
	Sold30d     int           `json:"sold30d"`
	Rating      float64       `json:"rating"`
	Hue         int           `json:"hue"`
	Trend       []int         `json:"trend"`
	Description string        `json:"description"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

func (p Product) Revenue30dCents() int64 { return int64(p.Sold30d) * p.PriceCents }

type ProductInput struct {
	Name        string        `json:"name"`
	SKU         string        `json:"sku"`
	Category    Category      `json:"category"`
	PriceCents  int64         `json:"priceCents"`
	Stock       int           `json:"stock"`
	Status      ProductStatus `json:"status"`
	Description string        `json:"description"`
}

func (in *ProductInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.SKU = strings.ToUpper(strings.TrimSpace(in.SKU))
	in.Description = strings.TrimSpace(in.Description)
}

func (in ProductInput) Validate() error {
	v := validator{}
	v.check(in.Name != "", "name", "is required")
	v.check(len(in.Name) <= 120, "name", "must be at most 120 characters")
	v.check(in.SKU != "", "sku", "is required")
	v.check(slices.Contains(Categories, in.Category), "category", "is not a known category")
	v.check(in.PriceCents >= 0, "priceCents", "must not be negative")
	v.check(in.Stock >= 0, "stock", "must not be negative")
	v.check(slices.Contains([]ProductStatus{ProductActive, ProductDraft, ProductArchived}, in.Status), "status", "must be active, draft or archived")
	return v.err()
}

type ProductFilter struct {
	Query    string
	Category Category
	Status   ProductStatus
	Sort     string // revenue | sales | price | stock | name
}

type ProductStats struct {
	Total               int              `json:"total"`
	Active              int              `json:"active"`
	LowStock            int              `json:"lowStock"`
	OutOfStock          int              `json:"outOfStock"`
	InventoryValueCents int64            `json:"inventoryValueCents"`
	ByCategory          map[Category]int `json:"byCategory"`
}

const LowStockThreshold = 15

type BulkAction string

const (
	BulkPublish BulkAction = "publish"
	BulkArchive BulkAction = "archive"
	BulkDelete  BulkAction = "delete"
)

// ── Orders & customers ──────────────────────────────────────────────────────

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

type OrderItem struct {
	ProductID  int64    `json:"productId"`
	Name       string   `json:"name"`
	SKU        string   `json:"sku"`
	Category   Category `json:"category"`
	Hue        int      `json:"hue"`
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
	Query  string
	Status OrderStatus
	Limit  int
}

type Customer struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Country    string    `json:"country"`
	Orders     int       `json:"orders"`
	LTVCents   int64     `json:"ltvCents"`
	Segment    string    `json:"segment"` // VIP | Regular | New | At risk
	LastSeenAt time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CustomerFilter struct {
	Query   string
	Segment string
}

type SegmentSummary struct {
	Count    int   `json:"count"`
	LTVCents int64 `json:"ltvCents"`
}

// ── Team ────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleEditor  Role = "editor"
	RoleSupport Role = "support"
	RoleViewer  Role = "viewer"
)

type MemberStatus string

const (
	MemberActive    MemberStatus = "active"
	MemberInvited   MemberStatus = "invited"
	MemberSuspended MemberStatus = "suspended"
)

type Member struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Role         Role         `json:"role"`
	Status       MemberStatus `json:"status"`
	MFA          bool         `json:"mfa"`
	LastActiveAt *time.Time   `json:"lastActiveAt"`
}

type MemberInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

func (in *MemberInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
}

func (in MemberInput) Validate() error {
	v := validator{}
	v.check(in.Name != "", "name", "is required")
	_, err := mail.ParseAddress(in.Email)
	v.check(err == nil, "email", "must be a valid email address")
	v.check(slices.Contains([]Role{RoleAdmin, RoleEditor, RoleSupport, RoleViewer}, in.Role), "role", "must be admin, editor, support or viewer")
	return v.err()
}

// ── Dashboard ───────────────────────────────────────────────────────────────

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
