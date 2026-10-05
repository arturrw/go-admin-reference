package httpapi

import (
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// Simulated storefront traffic. There is no real storefront behind this
// reference project, so visitors are a deterministic function of time: every
// poll sees the same history, and consecutive samples move smoothly. Product
// data (the hot product, its price, stock and 30-day sales) is real.

const (
	liveStep    = 2 * time.Second // one sample per step
	liveHistory = 60              // samples returned (two minutes)
)

type livePoint struct {
	At             time.Time `json:"at"`
	RequestsPerSec int       `json:"requestsPerSec"`
	OnlineUsers    int       `json:"onlineUsers"`
	HotViewers     int       `json:"hotViewers"`
}

type liveShare struct {
	Name string  `json:"name"`
	Pct  float64 `json:"pct"`
}

type livePage struct {
	Path    string `json:"path"`
	Viewers int    `json:"viewers"`
}

type hotProduct struct {
	d.TopProduct
	SKU               string  `json:"sku"`
	PriceCents        int64   `json:"priceCents"`
	Stock             int     `json:"stock"`
	Sold30d           int     `json:"sold30d"`
	Rating            float64 `json:"rating"`
	Viewers           int     `json:"viewers"`
	InCarts           int     `json:"inCarts"`
	SoldToday         int     `json:"soldToday"`
	RevenueTodayCents int64   `json:"revenueTodayCents"`
	ConversionPct     float64 `json:"conversionPct"`
	ShareOfTrafficPct float64 `json:"shareOfTrafficPct"`
}

type liveStats struct {
	At              time.Time   `json:"at"`
	RequestsPerSec  int         `json:"requestsPerSec"`
	OnlineUsers     int         `json:"onlineUsers"`
	ActiveCarts     int         `json:"activeCarts"`
	CheckoutsPerMin float64     `json:"checkoutsPerMin"`
	ConversionPct   float64     `json:"conversionPct"`
	AvgSessionSec   int         `json:"avgSessionSec"`
	BounceRatePct   float64     `json:"bounceRatePct"`
	History         []livePoint `json:"history"`
	Devices         []liveShare `json:"devices"`
	Sources         []liveShare `json:"sources"`
	TopPages        []livePage  `json:"topPages"`
	Hot             *hotProduct `json:"hot"`
}

// noise is a stable pseudo-random value in [-1, 1] for (key, slot).
func noise(key string, slot int64) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	var b [8]byte
	for i := range b {
		b[i] = byte(slot >> (8 * i))
	}
	_, _ = h.Write(b[:])
	return float64(h.Sum64()%20001)/10000 - 1
}

// wave is a smooth signal around 1: slow drift, a faster ripple and jitter.
func wave(key string, slot int64, amp float64) float64 {
	k := float64(slot)
	return 1 + amp*(.55*math.Sin(k/23+float64(len(key)))+.3*math.Sin(k/5.3)+.15*noise(key, slot))
}

func slotOf(t time.Time) int64 { return t.Unix() / int64(liveStep/time.Second) }

func liveRPS(slot int64) int    { return int(1240 * wave("rps", slot, .14)) }
func liveOnline(slot int64) int { return int(318 * wave("online", slot, .09)) }

// hotShare is the fraction of online shoppers looking at the hot product.
func hotShare(slot int64) float64 { return .074 * wave("hot", slot, .22) }

func (s *server) live(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	slot := slotOf(now)

	st := liveStats{
		At:              now,
		RequestsPerSec:  liveRPS(slot),
		OnlineUsers:     liveOnline(slot),
		ActiveCarts:     int(47 * wave("carts", slot, .12)),
		CheckoutsPerMin: math.Round(6.4*wave("checkout", slot, .25)*10) / 10,
		ConversionPct:   math.Round(3.84*wave("conv", slot/15, .05)*100) / 100,
		AvgSessionSec:   int(204 * wave("session", slot/15, .08)),
		BounceRatePct:   math.Round(38.5*wave("bounce", slot/15, .06)*10) / 10,
		History:         make([]livePoint, liveHistory),
	}
	for i := range st.History {
		sl := slot - int64(liveHistory-1-i)
		online := liveOnline(sl)
		st.History[i] = livePoint{
			At:             time.Unix(sl*int64(liveStep/time.Second), 0).UTC(),
			RequestsPerSec: liveRPS(sl),
			OnlineUsers:    online,
			HotViewers:     int(float64(online) * hotShare(sl)),
		}
	}

	mobile := 58 + 4*math.Sin(float64(slot)/40)
	st.Devices = []liveShare{{"Mobile", round1(mobile)}, {"Desktop", round1(92 - mobile)}, {"Tablet", 8}}
	organic := 34 + 3*math.Sin(float64(slot)/31)
	st.Sources = []liveShare{
		{"Organic search", round1(organic)}, {"Direct", 22}, {"Instagram", round1(52 - organic)},
		{"Newsletter", 9}, {"Referral", 7},
	}

	hot, err := s.hotProduct(r, now, slot, st.OnlineUsers)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	st.Hot = hot

	pages := []struct {
		path  string
		share float64
	}{{"/", .21}, {"/collections/new", .09}, {"/cart", .06}, {"/checkout", .03}}
	if hot != nil {
		st.TopPages = append(st.TopPages, livePage{Path: "/products/" + slug(hot.Name), Viewers: hot.Viewers})
	}
	for _, p := range pages {
		st.TopPages = append(st.TopPages, livePage{Path: p.path, Viewers: int(float64(st.OnlineUsers) * p.share * wave(p.path, slot, .2))})
	}
	writeJSON(w, http.StatusOK, st)
}

// hotCandidates is how many best sellers compete for "hottest right now".
const hotCandidates = 6

// hotProduct picks the product most viewed right now: the best sellers that
// are live and in stock compete, weighted by a demand signal that drifts
// minute to minute. So the pick follows the real catalogue (publish, archive,
// sell out, sales) and also changes over time. Today's sales grow with the
// time of day from the product's real 30-day rate.
func (s *server) hotProduct(r *http.Request, now time.Time, slot int64, online int) (*hotProduct, error) {
	items, err := s.store.ListProducts(r.Context(), d.ProductFilter{Status: d.ProductActive, Sort: "sales"})
	if err != nil {
		return nil, err
	}
	var p d.Product
	best := -1.0
	n := 0
	for _, c := range items {
		if c.Stock == 0 {
			continue
		}
		if score := float64(c.Sold30d) * wave(fmt.Sprintf("hot-%d", c.ID), slot/30, .45); score > best {
			p, best = c, score
		}
		if n++; n == hotCandidates {
			break
		}
	}
	if best < 0 {
		return nil, nil
	}
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayFrac := now.Sub(midnight).Hours() / 24
	soldToday := int(float64(p.Sold30d) / 30 * dayFrac)
	viewers := int(float64(online) * hotShare(slot))
	inCarts := max(1, int(float64(viewers)*.21*wave("hot-cart", slot, .2)))
	conv := 0.0
	if viewers > 0 {
		conv = math.Round(float64(inCarts)/float64(viewers)*.42*1000) / 10
	}
	return &hotProduct{
		TopProduct: d.TopProduct{
			ID: p.ID, Name: p.Name, Category: p.Category, Hue: p.Hue, ImageURL: p.ImageURL(),
			Sold: p.Sold30d, RevenueCents: p.Revenue30dCents(),
		},
		SKU: p.SKU, PriceCents: p.PriceCents, Stock: p.Stock, Sold30d: p.Sold30d, Rating: p.Rating,
		Viewers: viewers, InCarts: inCarts, SoldToday: soldToday, RevenueTodayCents: int64(soldToday) * p.PriceCents,
		ConversionPct: conv, ShareOfTrafficPct: round1(hotShare(slot) * 100),
	}, nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func slug(s string) string {
	out := make([]rune, 0, len(s))
	dash := false
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			out, dash = append(out, c), false
		case c >= 'A' && c <= 'Z':
			out, dash = append(out, c+32), false
		case !dash && len(out) > 0:
			out, dash = append(out, '-'), true
		}
	}
	if dash {
		out = out[:len(out)-1]
	}
	return string(out)
}
