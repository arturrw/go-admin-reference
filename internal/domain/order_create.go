package domain

import (
	"fmt"
	"slices"
	"time"
)

// PaymentMethods are what an order can be paid with.
var PaymentMethods = []string{"Visa •• 4242", "Mastercard •• 5100", "Apple Pay", "PayPal", "Amex •• 0005", "Google Pay"}

const (
	MaxOrderLines = 20
	MaxLineQty    = 99
)

// ConflictError is a request that is valid but clashes with the current state
// (not enough stock); the API answers 409.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

// NewOrderLine is one product and how many of it.
type NewOrderLine struct {
	ProductID int64 `json:"productId"`
	Qty       int   `json:"qty"`
}

// NewOrder is a sale entered by staff: who bought what and how they paid.
// Prices come from the catalogue at the moment of the sale.
type NewOrder struct {
	CustomerID int64          `json:"customerId"`
	Payment    string         `json:"payment"`
	Items      []NewOrderLine `json:"items"`
}

// Validate checks the shape and merges repeated products into one line.
func (in *NewOrder) Validate() error {
	v := validator{}
	v.check(in.CustomerID > 0, "customerId", "is required")
	v.check(slices.Contains(PaymentMethods, in.Payment), "payment", "is not an accepted payment method")
	items, err := ValidateLines(in.Items)
	if err != nil {
		for f, m := range err.(*ValidationError).Fields {
			v.check(false, f, m)
		}
	}
	if err := v.err(); err != nil {
		return err
	}
	in.Items = items
	return nil
}

// OrderEdit replaces the lines of an order that is still pending.
type OrderEdit struct {
	Items []NewOrderLine `json:"items"`
}

// ValidateLines checks product lines and merges repeated products.
func ValidateLines(items []NewOrderLine) ([]NewOrderLine, error) {
	v := validator{}
	v.check(len(items) > 0, "items", "add at least one product")
	v.check(len(items) <= MaxOrderLines, "items", fmt.Sprintf("at most %d products", MaxOrderLines))
	var merged []NewOrderLine
	for _, l := range items {
		v.check(l.ProductID > 0, "items", "every line needs a product")
		v.check(l.Qty >= 1 && l.Qty <= MaxLineQty, "items", fmt.Sprintf("quantity must be 1 to %d", MaxLineQty))
		if i := slices.IndexFunc(merged, func(m NewOrderLine) bool { return m.ProductID == l.ProductID }); i >= 0 {
			merged[i].Qty += l.Qty
			v.check(merged[i].Qty <= MaxLineQty, "items", fmt.Sprintf("quantity must be 1 to %d", MaxLineQty))
		} else {
			merged = append(merged, l)
		}
	}
	if err := v.err(); err != nil {
		return nil, err
	}
	return merged, nil
}

// EditedItems builds the lines of an edited order. A product already in the
// order keeps the price it was sold at; a new one sells at the catalogue price.
func EditedItems(cur []OrderItem, want []NewOrderLine, catalogue func(id int64) (Product, bool)) ([]OrderItem, error) {
	out := make([]OrderItem, 0, len(want))
	for _, l := range want {
		if i := slices.IndexFunc(cur, func(it OrderItem) bool { return it.ProductID == l.ProductID }); i >= 0 {
			it := cur[i]
			it.Qty = l.Qty
			out = append(out, it)
			continue
		}
		p, ok := catalogue(l.ProductID)
		if !ok {
			return nil, NewValidationError("items", fmt.Sprintf("product #%d does not exist", l.ProductID))
		}
		out = append(out, p.Line(l.Qty))
	}
	return out, nil
}

// StockDeltas is how many units each product's stock loses (positive) or gets
// back (negative) when an order's lines change from cur to next.
func StockDeltas(cur, next []OrderItem) map[int64]int {
	delta := map[int64]int{}
	for _, it := range next {
		delta[it.ProductID] += it.Qty
	}
	for _, it := range cur {
		delta[it.ProductID] -= it.Qty
	}
	for id, d := range delta {
		if d == 0 {
			delete(delta, id)
		}
	}
	return delta
}

// HasDeletedProduct is whether a line points at a product that no longer
// exists, which makes the order unsafe to re-edit.
func HasDeletedProduct(items []OrderItem) bool {
	return slices.ContainsFunc(items, func(it OrderItem) bool { return it.ProductID == 0 })
}

// CheckSellable reports why qty of the product cannot be sold now, or nil.
func CheckSellable(p Product, qty int) error {
	if p.Status != ProductActive {
		return NewValidationError("items", fmt.Sprintf("%s is not for sale", p.Name))
	}
	if p.Stock < qty {
		return &ConflictError{Message: fmt.Sprintf("only %d of %s in stock", p.Stock, p.Name)}
	}
	return nil
}

// Line snapshots the product at the price it sells for now.
func (p Product) Line(qty int) OrderItem {
	return OrderItem{
		ProductID: p.ID, Name: p.Name, SKU: p.SKU, Category: p.Category, Hue: p.Hue,
		ImageURL: p.ImageURL(), Qty: qty, PriceCents: p.PriceCents,
	}
}

// Total adds up the goods of an order's lines.
func Total(items []OrderItem) (cents int64) {
	for _, it := range items {
		cents += it.PriceCents * int64(it.Qty)
	}
	return cents
}

// PlacedEvent is the first entry of a new order's history.
func PlacedEvent(by string, at time.Time) OrderEvent {
	return OrderEvent{Status: OrderPending, At: at, By: by}
}
