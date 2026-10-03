// Package reqlog keeps a fixed-size ring buffer of recent HTTP requests that
// the admin UI shows on the "Request log" page.
package reqlog

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Time       time.Time `json:"time"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMs float64   `json:"durationMs"`
	Bytes      int       `json:"bytes"`
	RequestID  string    `json:"requestId"`
	IP         string    `json:"ip"`
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

// Recent returns up to limit entries, newest first, that satisfy keep (nil keeps all).
func (l *Log) Recent(limit int, keep func(Entry) bool) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := l.next
	if l.full {
		n = len(l.buf)
	}
	out := make([]Entry, 0, min(limit, n))
	for i := 1; i <= n && len(out) < limit; i++ {
		e := l.buf[(l.next-i+len(l.buf))%len(l.buf)]
		if keep == nil || keep(e) {
			out = append(out, e)
		}
	}
	return out
}

// Seed fills the buffer with plausible historical traffic so the page is not
// empty right after boot.
func (l *Log) Seed(now time.Time, n int) {
	r := rand.New(rand.NewPCG(7, 42))
	paths := map[string][]string{
		"GET":    {"/api/v1/products", "/api/v1/products/%d", "/api/v1/orders?limit=50", "/api/v1/customers", "/api/v1/dashboard?range=30", "/healthz", "/api/v1/team", "/api/v1/orders/%d"},
		"POST":   {"/api/v1/products", "/api/v1/products/bulk", "/api/v1/team"},
		"PUT":    {"/api/v1/products/%d", "/api/v1/team/%d"},
		"PATCH":  {"/api/v1/orders/%d/status"},
		"DELETE": {"/api/v1/products/%d", "/api/v1/team/%d"},
	}
	methods := []string{"GET", "GET", "GET", "GET", "GET", "GET", "POST", "POST", "PUT", "PATCH", "DELETE"}
	t := now.Add(-time.Duration(n) * 2300 * time.Millisecond)
	for range n {
		t = t.Add(time.Duration(400+r.IntN(3800)) * time.Millisecond)
		m := methods[r.IntN(len(methods))]
		p := paths[m][r.IntN(len(paths[m]))]
		if strings.Contains(p, "%") {
			p = fmt.Sprintf(p, 1+r.IntN(48))
		}
		status := 200
		if m == "POST" {
			status = 201
		}
		switch x := r.Float64(); {
		case x < .006:
			status = 500
		case x < .03:
			status = 404
		case x < .05:
			status = 422
		case x < .065:
			status = 401
		}
		ms := math.Pow(r.Float64(), 3)*380 + 2 + r.Float64()*12
		if status >= 500 {
			ms = 300 + r.Float64()*1500
		}
		if p == "/healthz" {
			ms = r.Float64() * 2
		}
		l.Add(Entry{
			Time: t, Method: m, Path: p, Status: status, DurationMs: math.Round(ms*10) / 10,
			Bytes: 200 + r.IntN(24000), RequestID: fmt.Sprintf("%08x", r.Uint32()),
			IP: fmt.Sprintf("10.0.%d.%d", r.IntN(10), 2+r.IntN(250)),
		})
	}
}
