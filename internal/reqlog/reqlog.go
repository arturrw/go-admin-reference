// Package reqlog keeps a fixed-size ring buffer of recent HTTP requests that
// the admin UI shows on the "Request log" page, plus aggregate analytics.
package reqlog

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Summary is the row shown in the list.
type Summary struct {
	ID         string    `json:"id"`
	Time       time.Time `json:"time"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMs float64   `json:"durationMs"`
	Bytes      int       `json:"bytes"`
	IP         string    `json:"ip"`
	Actor      string    `json:"actor"`
}

// Entry is the full record shown in the detail drawer. Sensitive headers
// and password fields are redacted before they are stored.
type Entry struct {
	Summary
	Query        string            `json:"query"`
	Proto        string            `json:"proto"`
	UserAgent    string            `json:"userAgent"`
	ActorRole    string            `json:"actorRole"`
	Route        string            `json:"route"` // matched pattern, e.g. "GET /api/v1/products/{id}"
	ReqHeaders   map[string]string `json:"requestHeaders"`
	RespHeaders  map[string]string `json:"responseHeaders"`
	ReqBody      string            `json:"requestBody"`
	RespBody     string            `json:"responseBody"` // captured for 4xx/5xx only
	ReqBytes     int64             `json:"requestBytes"`
	BodyTruncate bool              `json:"bodyTruncated"`
}

type Log struct {
	mu   sync.RWMutex
	buf  []Entry
	next int
	full bool
}

func New(size int) *Log { return &Log{buf: make([]Entry, size)} }

func (l *Log) Add(e Entry) {
	l.mu.Lock()
	l.buf[l.next] = e
	l.next = (l.next + 1) % len(l.buf)
	if l.next == 0 {
		l.full = true
	}
	l.mu.Unlock()
}

func (l *Log) each(fn func(Entry) bool) {
	n := l.next
	if l.full {
		n = len(l.buf)
	}
	for i := 1; i <= n; i++ {
		if !fn(l.buf[(l.next-i+len(l.buf))%len(l.buf)]) {
			return
		}
	}
}

// Recent returns up to limit summaries, newest first, that satisfy keep (nil keeps all).
func (l *Log) Recent(limit int, keep func(Entry) bool) []Summary {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []Summary{}
	l.each(func(e Entry) bool {
		if keep == nil || keep(e) {
			out = append(out, e.Summary)
		}
		return len(out) < limit
	})
	return out
}

func (l *Log) Get(id string) (Entry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var found Entry
	ok := false
	l.each(func(e Entry) bool {
		if e.ID == id {
			found, ok = e, true
			return false
		}
		return true
	})
	return found, ok
}

// ── Analytics ───────────────────────────────────────────────────────────────

type Endpoint struct {
	Route  string  `json:"route"`
	Count  int     `json:"count"`
	AvgMs  float64 `json:"avgMs"`
	P95Ms  float64 `json:"p95Ms"`
	Errors int     `json:"errors"`
}

type Stats struct {
	Total       int            `json:"total"`
	SuccessRate float64        `json:"successRate"`
	P50Ms       float64        `json:"p50Ms"`
	P95Ms       float64        `json:"p95Ms"`
	P99Ms       float64        `json:"p99Ms"`
	Client4xx   int            `json:"client4xx"`
	Server5xx   int            `json:"server5xx"`
	Methods     map[string]int `json:"methods"`
	PerMinute   []int          `json:"perMinute"` // last 30 minutes, oldest first
	Endpoints   []Endpoint     `json:"endpoints"` // top 8 by volume
}

var idSegment = regexp.MustCompile(`/(\d+|[0-9a-f]{8,})(/|$)`)

// RouteOf collapses ids so "/api/v1/products/12" groups as "/api/v1/products/{id}".
func RouteOf(method, path string) string {
	p, _, _ := strings.Cut(path, "?")
	for idSegment.MatchString(p) {
		p = idSegment.ReplaceAllString(p, "/{id}$2")
	}
	return method + " " + p
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[int(math.Round(float64(len(sorted)-1)*q))]
}

func (l *Log) Stats(now time.Time) Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	st := Stats{Methods: map[string]int{}, PerMinute: make([]int, 30)}
	var all []float64
	byRoute := map[string][]Entry{}
	ok := 0
	l.each(func(e Entry) bool {
		st.Total++
		all = append(all, e.DurationMs)
		st.Methods[e.Method]++
		switch {
		case e.Status < 400:
			ok++
		case e.Status < 500:
			st.Client4xx++
		default:
			st.Server5xx++
		}
		if m := int(now.Sub(e.Time).Minutes()); m >= 0 && m < 30 {
			st.PerMinute[29-m]++
		}
		r := e.Route
		if r == "" {
			r = RouteOf(e.Method, e.Path)
		}
		byRoute[r] = append(byRoute[r], e)
		return true
	})
	if st.Total == 0 {
		st.Endpoints = []Endpoint{}
		return st
	}
	slices.Sort(all)
	st.SuccessRate = math.Round(float64(ok)/float64(st.Total)*1000) / 10
	st.P50Ms, st.P95Ms, st.P99Ms = percentile(all, .5), percentile(all, .95), percentile(all, .99)

	for route, es := range byRoute {
		ds := make([]float64, len(es))
		ep := Endpoint{Route: route, Count: len(es)}
		var sum float64
		for i, e := range es {
			ds[i] = e.DurationMs
			sum += e.DurationMs
			if e.Status >= 400 {
				ep.Errors++
			}
		}
		slices.Sort(ds)
		ep.AvgMs = math.Round(sum/float64(len(es))*10) / 10
		ep.P95Ms = percentile(ds, .95)
		st.Endpoints = append(st.Endpoints, ep)
	}
	slices.SortFunc(st.Endpoints, func(a, b Endpoint) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Route, b.Route))
	})
	st.Endpoints = st.Endpoints[:min(8, len(st.Endpoints))]
	return st
}

// ── Seed ────────────────────────────────────────────────────────────────────

// Seed fills the buffer with plausible historical traffic so the page is not
// empty right after boot.
func (l *Log) Seed(now time.Time, n int) {
	r := rand.New(rand.NewPCG(7, 42))
	paths := map[string][]string{
		"GET":    {"/api/v1/products", "/api/v1/products/%d", "/api/v1/orders?limit=25", "/api/v1/customers", "/api/v1/customers/%d", "/api/v1/dashboard?range=30", "/healthz", "/api/v1/team", "/api/v1/orders/%d"},
		"POST":   {"/api/v1/products", "/api/v1/products/bulk", "/api/v1/team", "/api/v1/products/%d/images", "/api/v1/customers/%d/notes"},
		"PUT":    {"/api/v1/products/%d", "/api/v1/team/%d"},
		"PATCH":  {"/api/v1/orders/%d/status"},
		"DELETE": {"/api/v1/products/%d", "/api/v1/team/%d", "/api/v1/products/%d/images/gen-%d-1"},
	}
	actors := [][2]string{{"artur@acme.io", "owner"}, {"mark@acme.io", "admin"}, {"yuki@acme.io", "editor"}, {"priya@acme.io", "support"}, {"jon@acme.io", "viewer"}}
	agents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:142.0) Gecko/20100101 Firefox/142.0",
		"warehouse-sync/2.3 (+https://acme.io)",
	}
	methods := []string{"GET", "GET", "GET", "GET", "GET", "GET", "POST", "POST", "PUT", "PATCH", "DELETE"}
	// Spread the history over roughly the last 30 minutes.
	step := 30 * time.Minute / time.Duration(n)
	t := now.Add(-30 * time.Minute)
	for range n {
		t = t.Add(step/2 + time.Duration(r.Int64N(int64(step))))
		m := methods[r.IntN(len(methods))]
		p := paths[m][r.IntN(len(paths[m]))]
		for strings.Contains(p, "%d") {
			p = strings.Replace(p, "%d", strconv.Itoa(1+r.IntN(48)), 1)
		}
		status := 200
		switch m {
		case "POST":
			status = 201
		case "DELETE":
			status = 204
		}
		switch x := r.Float64(); {
		case x < .006:
			status = 500
		case x < .03:
			status = 404
		case x < .05:
			status = 422
		case x < .065:
			status = 403
		}
		ms := math.Pow(r.Float64(), 3)*380 + 2 + r.Float64()*12
		if status >= 500 {
			ms = 300 + r.Float64()*1500
		}
		if p == "/healthz" {
			ms = r.Float64() * 2
		}
		actor := actors[r.IntN(len(actors))]
		ua := agents[r.IntN(len(agents))]
		if p == "/healthz" {
			actor, ua = [2]string{"", ""}, "kube-probe/1.31"
		}
		path, query, _ := strings.Cut(p, "?")
		e := Entry{
			Summary: Summary{
				ID: fmt.Sprintf("%012x", r.Uint64()&0xffffffffffff), Time: t, Method: m, Path: p, Status: status,
				DurationMs: math.Round(ms*10) / 10, Bytes: 200 + r.IntN(24000),
				IP: fmt.Sprintf("10.0.%d.%d", r.IntN(10), 2+r.IntN(250)), Actor: actor[0],
			},
			Query: query, Proto: "HTTP/1.1", UserAgent: ua, ActorRole: actor[1], Route: RouteOf(m, path),
			ReqHeaders:  map[string]string{"Accept": "application/json", "User-Agent": ua, "Cookie": "[redacted]"},
			RespHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "X-Content-Type-Options": "nosniff"},
		}
		if m != "GET" && m != "DELETE" {
			e.ReqHeaders["Content-Type"] = "application/json"
			e.ReqBody = `{"status":"shipped"}`
			e.ReqBytes = int64(len(e.ReqBody))
		}
		if status >= 400 {
			e.RespBody = map[int]string{
				403: `{"error":"you don't have permission to do this"}`,
				404: `{"error":"not found"}`,
				422: `{"error":"validation failed","fields":{"name":"is required"}}`,
				500: `{"error":"internal error"}`,
			}[status]
		}
		l.Add(e)
	}
}
