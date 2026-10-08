package domain

import (
	"slices"
	"strings"
	"time"
)

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

// ProductImage is one picture in a product's gallery. The first image is the
// primary one shown in lists.
type ProductImage struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Alt       string `json:"alt"`
	Generated bool   `json:"generated"` // seed artwork or stock photo rather than an uploaded file
	SizeBytes int64  `json:"sizeBytes"`
}

const MaxProductImages = 8

type Product struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	SKU            string        `json:"sku"`
	Category       Category      `json:"category"`
	Vendor         string        `json:"vendor"`
	Tags           []string      `json:"tags"`
	PriceCents     int64         `json:"priceCents"`
	CompareAtCents int64         `json:"compareAtCents"` // 0 = not on sale
	CostCents      int64         `json:"costCents"`
	Stock          int           `json:"stock"`
	WeightGrams    int           `json:"weightGrams"`
	Status         ProductStatus `json:"status"`
	Sold30d        int           `json:"sold30d"`
	// Revenue30dCents and Trend come from orders (see ApplySales), not stored.
	Revenue30dCents int64          `json:"revenue30dCents"`
	SoldToday       int            `json:"-"` // UTC day so far
	RevenueToday    int64          `json:"-"`
	Rating          float64        `json:"rating"`
	Hue             int            `json:"hue"`
	Trend           []int          `json:"trend"`
	Description     string         `json:"description"`
	Images          []ProductImage `json:"images"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ImageURL returns the primary image or "" when the gallery is empty.
func (p Product) ImageURL() string {
	if len(p.Images) == 0 {
		return ""
	}
	return p.Images[0].URL
}

type ProductInput struct {
	Name           string        `json:"name"`
	SKU            string        `json:"sku"`
	Category       Category      `json:"category"`
	Vendor         string        `json:"vendor"`
	Tags           []string      `json:"tags"`
	PriceCents     int64         `json:"priceCents"`
	CompareAtCents int64         `json:"compareAtCents"`
	CostCents      int64         `json:"costCents"`
	Stock          int           `json:"stock"`
	WeightGrams    int           `json:"weightGrams"`
	Status         ProductStatus `json:"status"`
	Description    string        `json:"description"`
}

func (in *ProductInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.SKU = strings.ToUpper(strings.TrimSpace(in.SKU))
	in.Vendor = strings.TrimSpace(in.Vendor)
	in.Description = strings.TrimSpace(in.Description)
	in.Tags = normalizeTags(in.Tags)
}

func (in ProductInput) Validate() error {
	v := validator{}
	v.check(in.Name != "", "name", "is required")
	v.check(len(in.Name) <= 120, "name", "must be at most 120 characters")
	v.check(in.SKU != "", "sku", "is required")
	v.check(slices.Contains(Categories, in.Category), "category", "is not a known category")
	v.check(len(in.Vendor) <= 80, "vendor", "must be at most 80 characters")
	v.check(len(in.Tags) <= 10, "tags", "at most 10 tags")
	for _, t := range in.Tags {
		v.check(len(t) <= 24, "tags", "each tag must be at most 24 characters")
	}
	v.check(in.PriceCents >= 0, "priceCents", "must not be negative")
	v.check(in.CompareAtCents == 0 || in.CompareAtCents > in.PriceCents, "compareAtCents", "must be greater than the price (or empty)")
	v.check(in.CostCents >= 0, "costCents", "must not be negative")
	v.check(in.Stock >= 0, "stock", "must not be negative")
	v.check(in.WeightGrams >= 0, "weightGrams", "must not be negative")
	v.check(len(in.Description) <= 4000, "description", "must be at most 4000 characters")
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
