package memory

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// Fake data generation. Everything is seeded so every boot produces the same
// catalogue; timestamps are relative to the boot time so "x ago" stays fresh.

type catalogDef struct {
	hue        int
	minPrice   int64
	maxPrice   int64
	productSet []string
}

var catalog = map[d.Category]catalogDef{
	d.CategoryAudio:       {160, 39, 349, []string{"Aero Buds Pro", "Orbit Speaker Mini", "Bass Dome 360", "Studio Cans X", "Echo Bar Slim", "Aero Buds Lite"}},
	d.CategoryWearables:   {200, 59, 449, []string{"Pulse Watch S2", "Halo Ring Gen 3", "Stride Band 4", "Vital Clip", "Pulse Watch Ultra"}},
	d.CategoryLighting:    {45, 24, 189, []string{"Lumen Desk Lamp", "Arc Floor Light", "Glow Strip 5m", `Halo Ring Light 18"`, "Ember Bulb E27"}},
	d.CategoryHome:        {270, 29, 299, []string{"Nimbus Hub", "Thermo Dot", "Sentry Cam 2K", "Mist Diffuser", "Air Purifier One", "Smart Plug Duo"}},
	d.CategoryComputing:   {220, 19, 599, []string{"Flux Keyboard 75", "Drift Mouse", "Dock Prime 12-in-1", "Vertex Monitor Arm", "Nova SSD 2TB", "Flux Keyboard TKL", "Signal Webcam 4K"}},
	d.CategoryAccessories: {330, 12, 149, []string{"Atlas Backpack", "Vapor Bottle 750ml", "Cable Kit Braided", "Mag Wallet", `Folio Sleeve 14"`, "Grip Stand", "Travel Pouch"}},
}

var (
	firstNames = []string{"Anna", "Mark", "Sofia", "Jon", "Lena", "Omar", "Mia", "Lucas", "Yuki", "Noah", "Elif", "Mateo", "Chloe", "Ivan", "Priya", "Leo", "Hana", "Felix", "Aria", "Diego", "Nora", "Kai", "Zoe", "Arjun", "Maya", "Theo", "Ines", "Sam", "Ruth", "Pablo"}
	lastNames  = []string{"Petrova", "Liu", "Rossi", "Berg", "Kraft", "Haddad", "Novak", "Silva", "Tanaka", "Meyer", "Kaya", "Garcia", "Dubois", "Sokolov", "Shah", "Fischer", "Kim", "Wagner", "Costa", "Lopez", "Jensen", "Mori", "Laurent", "Patel", "Cohen", "Brooks", "Ruiz", "Ahmed", "Holm", "Vega"}
	domains    = []string{"gmail.com", "proton.me", "outlook.com", "fastmail.com", "icloud.com"}
	payments   = []string{"Visa •• 4242", "Mastercard •• 5100", "Apple Pay", "PayPal", "Amex •• 0005", "Google Pay"}
	markets    = []d.Market{{Country: "US", Name: "United States", SharePct: 31}, {Country: "DE", Name: "Germany", SharePct: 14}, {Country: "GB", Name: "United Kingdom", SharePct: 12}, {Country: "FR", Name: "France", SharePct: 9}, {Country: "NL", Name: "Netherlands", SharePct: 7}, {Country: "PL", Name: "Poland", SharePct: 6}, {Country: "CA", Name: "Canada", SharePct: 6}, {Country: "JP", Name: "Japan", SharePct: 5}, {Country: "SE", Name: "Sweden", SharePct: 4}, {Country: "BR", Name: "Brazil", SharePct: 3}, {Country: "ES", Name: "Spain", SharePct: 3}}
)

type weighted[T any] struct {
	v T
	w float64
}

func pickWeighted[T any](r *rand.Rand, items []weighted[T]) T {
	var total float64
	for _, it := range items {
		total += it.w
	}
	x := r.Float64() * total
	for _, it := range items {
		if x -= it.w; x <= 0 {
			return it.v
		}
	}
	return items[0].v
}

func between(r *rand.Rand, lo, hi int) int { return lo + r.IntN(hi-lo+1) }

func seed(s *Store, now time.Time) {
	r := rand.New(rand.NewPCG(2025, 1003))

	// Products.
	var id int64
	for _, cat := range d.Categories {
		def := catalog[cat]
		for _, name := range def.productSet {
			id++
			price := def.minPrice + r.Int64N(def.maxPrice-def.minPrice+1)
			price = (price/5)*5*100 - 1 // e.g. 12499 → $124.99
			stock := between(r, 20, 640)
			switch x := r.Float64(); {
			case x < .08:
				stock = 0
			case x < .2:
				stock = between(r, 1, d.LowStockThreshold-1)
			}
			trend := make([]int, 14)
			slope := between(r, -1, 3)
			for i := range trend {
				trend[i] = max(1, 40+slope*i+int((r.Float64()-.5)*30))
			}
			s.products = append(s.products, d.Product{
				ID:          id,
				Name:        name,
				SKU:         fmt.Sprintf("%s-%04d", strings.ToUpper(string(cat)[:3]), between(r, 1000, 9999)),
				Category:    cat,
				PriceCents:  price,
				Stock:       stock,
				Status:      pickWeighted(r, []weighted[d.ProductStatus]{{d.ProductActive, 76}, {d.ProductDraft, 14}, {d.ProductArchived, 10}}),
				Sold30d:     between(r, 18, 920),
				Rating:      math.Round((3.6+r.Float64()*1.3)*10) / 10,
				Hue:         def.hue + between(r, -18, 18),
				Trend:       trend,
				Description: "Designed for everyday use. Premium materials, 2-year warranty, ships in recycled packaging.",
				UpdatedAt:   now.Add(-time.Duration(between(r, 1, 40*24)) * time.Hour),
			})
		}
	}
	s.nextProductID = id + 1

	// Customers.
	marketWeights := make([]weighted[string], len(markets))
	for i, m := range markets {
		marketWeights[i] = weighted[string]{m.Country, m.SharePct}
	}
	for i := range 42 {
		first, last := firstNames[(i*7)%len(firstNames)], lastNames[(i*11+3)%len(lastNames)]
		orders := between(r, 1, 38)
		segment := "Regular"
		switch {
		case orders > 24:
			segment = "VIP"
		case orders < 4:
			segment = "New"
		case r.Float64() < .2:
			segment = "At risk"
		}
		s.customers = append(s.customers, d.Customer{
			ID:         int64(i + 1),
			Name:       first + " " + last,
			Email:      strings.ToLower(first+"."+last) + "@" + domains[r.IntN(len(domains))],
			Country:    pickWeighted(r, marketWeights),
			Orders:     orders,
			LTVCents:   int64(float64(orders) * (40 + r.Float64()*160) * 100),
			Segment:    segment,
			LastSeenAt: now.Add(-time.Duration(math.Pow(r.Float64(), 3)*20*24*60) * time.Minute),
			CreatedAt:  now.AddDate(-between(r, 0, 3), -between(r, 0, 11), 0),
		})
	}

	// Orders: newest first, spaced increasingly further apart.
	for i := range 64 {
		c := s.customers[r.IntN(len(s.customers))]
		var items []d.OrderItem
		var total int64
		for range between(r, 1, 4) {
			p := s.products[r.IntN(len(s.products))]
			qty := between(r, 1, 3)
			items = append(items, d.OrderItem{ProductID: p.ID, Name: p.Name, SKU: p.SKU, Category: p.Category, Hue: p.Hue, Qty: qty, PriceCents: p.PriceCents})
			total += p.PriceCents * int64(qty)
		}
		status := d.OrderPending
		if i >= 4 {
			status = pickWeighted(r, []weighted[d.OrderStatus]{{d.OrderPaid, 34}, {d.OrderShipped, 30}, {d.OrderDelivered, 22}, {d.OrderPending, 8}, {d.OrderRefunded, 4}, {d.OrderFailed, 2}})
		}
		s.orders = append(s.orders, d.Order{
			ID:         int64(10480 - i),
			Customer:   d.CustomerRef{ID: c.ID, Name: c.Name, Email: c.Email, Country: c.Country, Segment: c.Segment},
			Items:      items,
			TotalCents: total,
			Status:     status,
			Payment:    payments[r.IntN(len(payments))],
			PlacedAt:   now.Add(-time.Duration(math.Pow(float64(i), 1.7)*6+2) * time.Minute),
		})
	}

	// Team.
	ago := func(m int) *time.Time { t := now.Add(-time.Duration(m) * time.Minute); return &t }
	s.members = []d.Member{
		{ID: 1, Name: "Anna Petrova", Email: "anna@acme.io", Role: d.RoleOwner, Status: d.MemberActive, MFA: true, LastActiveAt: ago(0)},
		{ID: 2, Name: "Mark Liu", Email: "mark@acme.io", Role: d.RoleAdmin, Status: d.MemberActive, MFA: true, LastActiveAt: ago(12)},
		{ID: 3, Name: "Sofia Rossi", Email: "sofia@acme.io", Role: d.RoleEditor, Status: d.MemberInvited, MFA: false, LastActiveAt: nil},
		{ID: 4, Name: "Jon Berg", Email: "jon@acme.io", Role: d.RoleViewer, Status: d.MemberActive, MFA: true, LastActiveAt: ago(180)},
		{ID: 5, Name: "Lena Kraft", Email: "lena@acme.io", Role: d.RoleViewer, Status: d.MemberSuspended, MFA: false, LastActiveAt: ago(14 * 24 * 60)},
		{ID: 6, Name: "Omar Haddad", Email: "omar@acme.io", Role: d.RoleAdmin, Status: d.MemberActive, MFA: true, LastActiveAt: ago(60)},
		{ID: 7, Name: "Yuki Tanaka", Email: "yuki@acme.io", Role: d.RoleEditor, Status: d.MemberActive, MFA: true, LastActiveAt: ago(22)},
		{ID: 8, Name: "Diego Vega", Email: "diego@acme.io", Role: d.RoleSupport, Status: d.MemberActive, MFA: false, LastActiveAt: ago(26 * 60)},
		{ID: 9, Name: "Priya Shah", Email: "priya@acme.io", Role: d.RoleSupport, Status: d.MemberActive, MFA: true, LastActiveAt: ago(5)},
	}
	s.nextMemberID = 10

	// Daily revenue for the last 90 days plus the preceding period.
	const days = 90
	for i := range days {
		day := now.AddDate(0, 0, -(days - 1 - i))
		weekend := day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
		v := 6200 + float64(i)*48 + math.Sin(float64(i)/6)*900 + (r.Float64()-.5)*1800
		if weekend {
			v -= 1100
		} else {
			v += 600
		}
		prev := v*.82 + math.Sin(float64(i)/5)*700 + (r.Float64()-.5)*1400
		s.revenue = append(s.revenue, d.RevenuePoint{Date: day.Format(time.DateOnly), Current: int64(v * 100), Previous: int64(prev * 100)})
	}

	// Orders heatmap: midday peak plus an evening bump, quieter weekends.
	for day := range 7 {
		for h := range 24 {
			x := math.Exp(-math.Pow((float64(h)-13.5)/4.5, 2)) + .55*math.Exp(-math.Pow((float64(h)-20.5)/2, 2))
			if day >= 5 {
				x *= .7
			}
			x = math.Max(0, math.Min(1, x+(r.Float64()-.5)*.22))
			s.heatmap[day][h] = int(x * 140)
		}
	}

	s.activity = []d.Activity{
		{Kind: "role", Actor: "Mark Liu", Message: "changed Sofia Rossi's role to editor", At: now.Add(-14 * time.Minute)},
		{Kind: "publish", Actor: "Yuki Tanaka", Message: "published Pulse Watch Ultra", At: now.Add(-38 * time.Minute)},
		{Kind: "deploy", Actor: "CI", Message: "rolled out v1.4.2 to 3/3 pods", At: now.Add(-2 * time.Hour)},
		{Kind: "stock", Actor: "Inventory", Message: "Glow Strip 5m dropped below 10 units", At: now.Add(-3 * time.Hour)},
		{Kind: "refund", Actor: "Priya Shah", Message: "issued a $129.99 refund for #10431", At: now.Add(-5 * time.Hour)},
	}
}
