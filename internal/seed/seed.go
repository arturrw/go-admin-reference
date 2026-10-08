// Package seed generates the deterministic fake dataset used by every store.
package seed

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/media"
)

// Fake data generation. Everything is seeded so every boot produces the same
// catalogue; timestamps are relative to the boot time so "x ago" stays fresh.

// DemoPassword is the password of every seeded, active team member.
const DemoPassword = "goadmin"

type catalogDef struct {
	hue        int
	minPrice   int64
	maxPrice   int64
	weight     [2]int // grams
	vendors    []string
	tags       []string
	productSet []string
}

var catalog = map[d.Category]catalogDef{
	d.CategoryAudio:       {160, 39, 349, [2]int{40, 900}, []string{"Aero Labs", "Sonic Forge"}, []string{"wireless", "bluetooth", "noise-cancelling", "bestseller"}, []string{"Aero Buds Pro", "Orbit Speaker Mini", "Bass Dome 360", "Studio Cans X", "Echo Bar Slim", "Aero Buds Lite"}},
	d.CategoryWearables:   {200, 59, 449, [2]int{20, 120}, []string{"Pulse", "Halo Health"}, []string{"fitness", "waterproof", "gps", "new"}, []string{"Pulse Watch S2", "Halo Ring Gen 3", "Stride Band 4", "Vital Clip", "Pulse Watch Ultra"}},
	d.CategoryLighting:    {45, 24, 189, [2]int{150, 4200}, []string{"Lumen & Co", "Arc Studio"}, []string{"led", "dimmable", "smart-home", "warm-white"}, []string{"Lumen Desk Lamp", "Arc Floor Light", "Glow Strip 5m", `Halo Ring Light 18"`, "Ember Bulb E27"}},
	d.CategoryHome:        {270, 29, 299, [2]int{90, 3800}, []string{"Nimbus", "Hearth Labs"}, []string{"smart-home", "matter", "energy-saving", "gift"}, []string{"Nimbus Hub", "Thermo Dot", "Sentry Cam 2K", "Mist Diffuser", "Air Purifier One", "Smart Plug Duo"}},
	d.CategoryComputing:   {220, 19, 599, [2]int{60, 2600}, []string{"Flux Devices", "Vertex"}, []string{"usb-c", "mechanical", "ergonomic", "pro"}, []string{"Flux Keyboard 75", "Drift Mouse", "Dock Prime 12-in-1", "Vertex Monitor Arm", "Nova SSD 2TB", "Flux Keyboard TKL", "Signal Webcam 4K"}},
	d.CategoryAccessories: {330, 12, 149, [2]int{30, 1400}, []string{"Atlas Goods", "Fold & Co"}, []string{"travel", "recycled", "vegan-leather", "gift"}, []string{"Atlas Backpack", "Vapor Bottle 750ml", "Cable Kit Braided", "Mag Wallet", `Folio Sleeve 14"`, "Grip Stand", "Travel Pouch"}},
}

var (
	firstNames = []string{"Anna", "Mark", "Sofia", "Jon", "Lena", "Omar", "Mia", "Lucas", "Yuki", "Noah", "Elif", "Mateo", "Chloe", "Ivan", "Priya", "Leo", "Hana", "Felix", "Aria", "Diego", "Nora", "Kai", "Zoe", "Arjun", "Maya", "Theo", "Ines", "Sam", "Ruth", "Pablo"}
	lastNames  = []string{"Petrova", "Liu", "Rossi", "Berg", "Kraft", "Haddad", "Novak", "Silva", "Tanaka", "Meyer", "Kaya", "Garcia", "Dubois", "Sokolov", "Shah", "Fischer", "Kim", "Wagner", "Costa", "Lopez", "Jensen", "Mori", "Laurent", "Patel", "Cohen", "Brooks", "Ruiz", "Ahmed", "Holm", "Vega"}
	domains    = []string{"gmail.com", "proton.me", "outlook.com", "fastmail.com", "icloud.com"}
	payments   = []string{"Visa •• 4242", "Mastercard •• 5100", "Apple Pay", "PayPal", "Amex •• 0005", "Google Pay"}
	sources    = []string{"Organic search", "Instagram", "Referral", "Newsletter", "Google Ads", "Direct"}
	custTags   = []string{"newsletter", "early-adopter", "gift-buyer", "wholesale", "support-escalation", "beta-tester"}
	streets    = []string{"Market St", "Linden Ave", "Harbour Rd", "Maple Way", "King St", "Station Rd", "Park Lane", "Elm Row"}
	markets    = []d.Market{
		{Country: "US", Name: "United States", SharePct: 31}, {Country: "DE", Name: "Germany", SharePct: 14},
		{Country: "GB", Name: "United Kingdom", SharePct: 12}, {Country: "FR", Name: "France", SharePct: 9},
		{Country: "NL", Name: "Netherlands", SharePct: 7}, {Country: "PL", Name: "Poland", SharePct: 6},
		{Country: "CA", Name: "Canada", SharePct: 6}, {Country: "JP", Name: "Japan", SharePct: 5},
		{Country: "SE", Name: "Sweden", SharePct: 4}, {Country: "BR", Name: "Brazil", SharePct: 3},
		{Country: "ES", Name: "Spain", SharePct: 3},
	}
	cities = map[string][]string{
		"US": {"San Francisco", "Austin", "Brooklyn"}, "DE": {"Berlin", "Munich", "Hamburg"}, "GB": {"London", "Manchester", "Bristol"},
		"FR": {"Paris", "Lyon"}, "NL": {"Amsterdam", "Utrecht"}, "PL": {"Warsaw", "Kraków"}, "CA": {"Toronto", "Vancouver"},
		"JP": {"Tokyo", "Osaka"}, "SE": {"Stockholm"}, "BR": {"São Paulo"}, "ES": {"Madrid", "Barcelona"},
	}
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

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func between(r *rand.Rand, lo, hi int) int { return lo + r.IntN(hi-lo+1) }

func generatedImages(p d.Product) []d.ProductImage {
	imgs := make([]d.ProductImage, media.ShotCount())
	for i := range imgs {
		imgs[i] = d.ProductImage{
			ID:        fmt.Sprintf("gen-%d-%d", p.ID, i),
			URL:       fmt.Sprintf("/media/generated/%d/%d.svg?c=%s&h=%d", p.ID, i, p.Category, p.Hue),
			Alt:       p.Name + " — " + media.ShotLabel(i),
			Generated: true,
		}
	}
	return imgs
}

// Dataset is everything a fresh store starts with.
type Dataset struct {
	Products  []d.Product
	Customers []d.Customer
	Orders    []d.Order // newest first
	Members   []d.Member
	Activity  []d.Activity
	APIKeys   []d.APIKey
}

// CategoryHue is the base artwork hue of a category.
func CategoryHue(c d.Category) int { return catalog[c].hue }

// Generate builds the dataset; timestamps are relative to now.
func Generate(now time.Time) *Dataset {
	s := &Dataset{}
	r := rand.New(rand.NewPCG(2025, 1003))

	// ── Products ──
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
			var compareAt int64
			if r.Float64() < .25 {
				compareAt = price + int64(between(r, 2, 8))*500
			}
			tags := []string{pick(r, def.tags)}
			if t := pick(r, def.tags); t != tags[0] {
				tags = append(tags, t)
			}
			p := d.Product{
				ID:             id,
				Name:           name,
				SKU:            fmt.Sprintf("%s-%04d", strings.ToUpper(string(cat)[:3]), between(r, 1000, 9999)),
				Category:       cat,
				Vendor:         pick(r, def.vendors),
				Tags:           tags,
				PriceCents:     price,
				CompareAtCents: compareAt,
				CostCents:      price * int64(between(r, 32, 58)) / 100,
				Stock:          stock,
				WeightGrams:    between(r, def.weight[0], def.weight[1]),
				Status:         pickWeighted(r, []weighted[d.ProductStatus]{{d.ProductActive, 76}, {d.ProductDraft, 14}, {d.ProductArchived, 10}}),
				Sold30d:        between(r, 18, 920),
				Rating:         math.Round((3.6+r.Float64()*1.3)*10) / 10,
				Hue:            def.hue + between(r, -18, 18),
				Trend:          trend,
				Description:    fmt.Sprintf("%s by %s. Designed for everyday use with premium materials, a 2-year warranty and recycled packaging.", name, "the studio"),
				CreatedAt:      now.AddDate(0, -between(r, 2, 20), 0),
				UpdatedAt:      now.Add(-time.Duration(between(r, 1, 40*24)) * time.Hour),
			}
			p.Description = strings.Replace(p.Description, "the studio", p.Vendor, 1)
			p.Images = seedImages(p)
			s.Products = append(s.Products, p)
		}
	}

	// ── Customers ──
	marketWeights := make([]weighted[string], len(markets))
	for i, m := range markets {
		marketWeights[i] = weighted[string]{m.Country, m.SharePct}
	}
	staff := []string{"Mark Liu", "Priya Shah", "Diego Vega", "Yuki Tanaka"}
	noteTexts := []string{
		"Asked about bulk pricing for an office order.",
		"Package arrived damaged — sent a replacement, no return needed.",
		"Prefers email over phone. Very responsive.",
		"Interested in the upcoming Pulse Watch release.",
		"Requested invoice with VAT number for the company.",
		"Left a 5★ review after support call.",
	}
	var noteID int64
	const customerCount = 151
	for i := range customerCount {
		// The pair is unique for the first len(first)*len(last) customers.
		first, last := firstNames[(i*7)%len(firstNames)], lastNames[(i*11+3+i/len(firstNames))%len(lastNames)]
		country := pickWeighted(r, marketWeights)
		c := d.Customer{
			ID:      int64(i + 1),
			Name:    first + " " + last,
			Email:   strings.ToLower(first+"."+last) + "@" + pick(r, domains),
			Phone:   fmt.Sprintf("+%d %03d %03d %04d", between(r, 1, 49), between(r, 100, 999), between(r, 100, 999), between(r, 0, 9999)),
			Country: country,
			Address: d.Address{
				Line1:      fmt.Sprintf("%d %s", between(r, 2, 240), pick(r, streets)),
				City:       pick(r, cities[country]),
				PostalCode: fmt.Sprintf("%05d", between(r, 1000, 99999)),
				Country:    country,
			},
			AcceptsMarketing: r.Float64() < .62,
			Source:           pick(r, sources),
			LastSeenAt:       now.Add(-time.Duration(math.Pow(r.Float64(), 3)*20*24*60) * time.Minute),
		}
		for range between(r, 0, 2) {
			if t := pick(r, custTags); !slices.Contains(c.Tags, t) {
				c.Tags = append(c.Tags, t)
			}
		}
		for k := range between(r, 0, 2) {
			noteID++
			c.Notes = append(c.Notes, d.CustomerNote{
				ID: noteID, Author: pick(r, staff), Text: pick(r, noteTexts),
				At: now.Add(-time.Duration(between(r, 2, 60*24)+k*90) * time.Hour),
			})
		}
		if c.Tags == nil {
			c.Tags = []string{}
		}
		if c.Notes == nil {
			c.Notes = []d.CustomerNote{}
		}
		s.Customers = append(s.Customers, c)
	}

	// ── Orders: each customer gets a purchase history; timestamps are
	// assigned newest-first with gaps that grow into the past. ──
	var owners []int
	for i := range s.Customers {
		n := pickWeighted(r, []weighted[int]{{1, 18}, {2, 18}, {3, 16}, {5, 18}, {8, 16}, {13, 10}, {18, 4}})
		for range n {
			owners = append(owners, i)
		}
	}
	r.Shuffle(len(owners), func(i, j int) { owners[i], owners[j] = owners[j], owners[i] })
	const recent = 40
	for i, ci := range owners {
		c := s.Customers[ci]
		// The newest orders are dense (the first ~3 days), the rest are spread
		// evenly over six months so the dashboard has a history to compare.
		mins := math.Pow(float64(i), 1.75)*7 + 2
		if i >= recent {
			head := math.Pow(recent, 1.75)*7 + 2
			// Exponent above 1: sales thin out into the past, so the business grows.
			mins = head + math.Pow(float64(i-recent)/float64(len(owners)-recent), 1.15)*(180*24*60-head)
		}
		placed := now.Add(-time.Duration(mins) * time.Minute)
		age := now.Sub(placed)
		var status d.OrderStatus
		switch {
		case i < 4:
			status = d.OrderPending
		case age < 6*time.Hour:
			status = pickWeighted(r, []weighted[d.OrderStatus]{{d.OrderPending, 3}, {d.OrderPaid, 7}})
		case age < 3*24*time.Hour:
			status = pickWeighted(r, []weighted[d.OrderStatus]{{d.OrderPaid, 4}, {d.OrderShipped, 6}, {d.OrderFailed, .3}})
		default:
			status = pickWeighted(r, []weighted[d.OrderStatus]{{d.OrderDelivered, 90}, {d.OrderShipped, 3}, {d.OrderRefunded, 5}, {d.OrderFailed, 2}})
		}
		o := d.Order{ID: int64(10000 + len(owners) - i), Status: status, Payment: pick(r, payments), PlacedAt: placed}
		for range between(r, 1, 4) {
			p := s.Products[r.IntN(len(s.Products))]
			qty := pickWeighted(r, []weighted[int]{{1, 8}, {2, 2}, {3, 1}})
			o.Items = append(o.Items, d.OrderItem{ProductID: p.ID, Name: p.Name, SKU: p.SKU, Category: p.Category, Hue: p.Hue, ImageURL: p.ImageURL(), Qty: qty, PriceCents: p.PriceCents})
			o.TotalCents += p.PriceCents * int64(qty)
		}
		o.Customer = c.Ref() // refreshed after segments are computed
		s.Orders = append(s.Orders, o)
	}
	for i := range s.Customers {
		c := &s.Customers[i]
		d.DeriveCustomer(c, s.Orders, now)
		c.CreatedAt = c.LastOrderAt
		for _, o := range s.Orders {
			if o.Customer.ID == c.ID && o.PlacedAt.Before(c.CreatedAt) {
				c.CreatedAt = o.PlacedAt
			}
		}
		// Signing up shortly before the first order keeps "new customers" real.
		c.CreatedAt = c.CreatedAt.Add(-time.Duration(between(r, 1, 48)) * time.Hour)
		if c.LastSeenAt.Before(c.LastOrderAt) {
			c.LastSeenAt = c.LastOrderAt
		}
		d.DeriveCustomer(c, s.Orders, now) // segment depends on CreatedAt
	}
	for i := range s.Orders {
		s.Orders[i].Customer = s.Customers[s.Orders[i].Customer.ID-1].Ref()
	}
	slices.SortFunc(s.Orders, func(a, b d.Order) int { return cmp.Compare(b.ID, a.ID) })
	refunders := []string{"Priya Shah", "Diego Vega"}
	refunds := 0
	for i := range s.Orders {
		if o := &s.Orders[i]; o.Status == d.OrderRefunded {
			at := o.PlacedAt.Add(time.Duration(between(r, 20, 96)) * time.Hour)
			if at.After(now) {
				at = now.Add(-time.Hour)
			}
			o.Refund = &d.OrderRefund{Reason: d.RefundReasons[refunds%len(d.RefundReasons)], By: refunders[refunds%len(refunders)], At: at}
			refunds++
		}
	}

	// ── Team ──
	hash, err := auth.HashPassword(DemoPassword)
	if err != nil {
		panic(err)
	}
	ago := func(m int) *time.Time { t := now.Add(-time.Duration(m) * time.Minute); return &t }
	s.Members = []d.Member{
		{ID: 1, Name: "Artur DCS", Email: "artur@acme.io", Role: d.RoleOwner, Status: d.MemberActive, MFA: true, LastActiveAt: ago(0)},
		{ID: 2, Name: "Mark Liu", Email: "mark@acme.io", Role: d.RoleAdmin, Status: d.MemberActive, MFA: true, LastActiveAt: ago(12)},
		{ID: 3, Name: "Sofia Rossi", Email: "sofia@acme.io", Role: d.RoleEditor, Status: d.MemberInvited, MFA: false, LastActiveAt: nil},
		{ID: 4, Name: "Jon Berg", Email: "jon@acme.io", Role: d.RoleViewer, Status: d.MemberActive, MFA: true, LastActiveAt: ago(180)},
		{ID: 5, Name: "Lena Kraft", Email: "lena@acme.io", Role: d.RoleViewer, Status: d.MemberSuspended, MFA: false, LastActiveAt: ago(14 * 24 * 60)},
		{ID: 6, Name: "Omar Haddad", Email: "omar@acme.io", Role: d.RoleAdmin, Status: d.MemberActive, MFA: true, LastActiveAt: ago(60),
			Revoked: []d.Permission{d.PermSettingsWrite}},
		{ID: 7, Name: "Yuki Tanaka", Email: "yuki@acme.io", Role: d.RoleEditor, Status: d.MemberActive, MFA: true, LastActiveAt: ago(22)},
		{ID: 8, Name: "Diego Vega", Email: "diego@acme.io", Role: d.RoleSupport, Status: d.MemberActive, MFA: false, LastActiveAt: ago(26 * 60),
			Granted: []d.Permission{d.PermProductsWrite}},
		{ID: 9, Name: "Priya Shah", Email: "priya@acme.io", Role: d.RoleSupport, Status: d.MemberActive, MFA: true, LastActiveAt: ago(5)},
	}
	for i := range s.Orders {
		s.Orders[i].Events = OrderEvents(s.Orders[i], now)
	}

	for i := range s.Members {
		m := &s.Members[i]
		if m.Granted == nil {
			m.Granted = []d.Permission{}
		}
		if m.Revoked == nil {
			m.Revoked = []d.Permission{}
		}
		if s.Members[i].Status != d.MemberInvited {
			// One shared hash keeps boot fast; real members get their own salt.
			s.Members[i].PasswordHash = hash
		}
	}

	s.Activity = BuildActivity(s, now)
	s.APIKeys = APIKeys(now)
	return s
}
