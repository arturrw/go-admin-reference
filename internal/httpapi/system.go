package httpapi

import (
	"context"
	"math"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/media"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
)

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if p, ok := s.store.(Pinger); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "database": "down"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) meta(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.ProductStats(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	counts, err := s.store.OrderCounts(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":       s.version,
		"serviceName":   s.settings(r.Context()).ServiceName,
		"env":           s.env,
		"goVersion":     runtime.Version(),
		"products":      stats.Total,
		"pendingOrders": counts[domain.OrderPending],
	})
}

type runtimeStats struct {
	Goroutines    int     `json:"goroutines"`
	HeapAllocMB   float64 `json:"heapAllocMb"`
	HeapSysMB     float64 `json:"heapSysMb"`
	NumGC         uint32  `json:"numGc"`
	GCPauseMaxMs  float64 `json:"gcPauseMaxMs"`
	GOMAXPROCS    int     `json:"gomaxprocs"`
	UptimeSeconds int64   `json:"uptimeSeconds"`
	GoVersion     string  `json:"goVersion"`
	Platform      string  `json:"platform"`
	Revision      string  `json:"revision"`
	// Simulated storefront traffic (see live.go).
	RequestsPerSec int `json:"requestsPerSec"`
	OnlineUsers    int `json:"onlineUsers"`
}

func (s *server) runtime(w http.ResponseWriter, _ *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	gc := debug.GCStats{PauseQuantiles: make([]time.Duration, 5)}
	debug.ReadGCStats(&gc)

	rev := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			if kv.Key == "vcs.revision" && len(kv.Value) >= 7 {
				rev = kv.Value[:7]
			}
		}
	}
	mb := func(b uint64) float64 { return math.Round(float64(b)/(1<<20)*10) / 10 }
	writeJSON(w, http.StatusOK, runtimeStats{
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocMB:    mb(ms.HeapAlloc),
		HeapSysMB:      mb(ms.HeapSys),
		NumGC:          ms.NumGC,
		GCPauseMaxMs:   float64(gc.PauseQuantiles[4].Microseconds()) / 1000,
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
		UptimeSeconds:  int64(time.Since(s.started).Seconds()),
		GoVersion:      runtime.Version(),
		Platform:       runtime.GOOS + "/" + runtime.GOARCH,
		Revision:       rev,
		RequestsPerSec: liveRPS(slotOf(time.Now())),
		OnlineUsers:    liveOnline(slotOf(time.Now())),
	})
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	dash, err := s.store.Dashboard(r.Context(), queryInt(r, "range", 30))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

// setTarget: PUT /api/v1/target {goalCents} — the current quarter's revenue
// goal. Owner only (workspace:manage).
func (s *server) setTarget(w http.ResponseWriter, r *http.Request) {
	var in struct {
		GoalCents int64 `json:"goalCents"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := domain.ValidateGoal(in.GoalCents); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	now := time.Now()
	key, _, _ := domain.QuarterOf(now)
	prev, err := s.store.TargetGoal(r.Context(), key)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	me, _ := CurrentMember(r.Context())
	g, err := s.store.SetTargetGoal(r.Context(), domain.TargetGoal{Quarter: key, GoalCents: in.GoalCents, UpdatedBy: me.Name})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	before := int64(domain.DefaultGoalCents)
	if prev != nil {
		before = prev.GoalCents
	}
	t := domain.BuildTarget(now, &g)
	s.audit(r.Context(), domain.ActTarget, "", 0, "set the %s target to %s (was %s)", key[5:]+" "+key[:4], domain.USD(g.GoalCents), domain.USD(before))
	writeJSON(w, http.StatusOK, t)
}

// listRequests supports ?limit=, ?class=2|4|5, ?method=, ?actor= and ?q=
// (substring of method, path, status or actor).
func (s *server) listRequests(w http.ResponseWriter, r *http.Request) {
	qv := r.URL.Query()
	limit := min(max(queryInt(r, "limit", 120), 1), 500)
	class, method, actor := qv.Get("class"), strings.ToUpper(qv.Get("method")), qv.Get("actor")
	q := strings.ToLower(qv.Get("q"))

	items := s.requests.Recent(limit, func(e reqlog.Entry) bool {
		status := strconv.Itoa(e.Status)
		switch {
		case class != "" && !strings.HasPrefix(status, class),
			method != "" && e.Method != method,
			actor != "" && e.Actor != actor:
			return false
		}
		return q == "" || strings.Contains(strings.ToLower(e.Method+" "+e.Path+" "+status+" "+e.Actor), q)
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "stats": s.requests.Stats(time.Now())})
}

func (s *server) getRequest(w http.ResponseWriter, r *http.Request) {
	e, ok := s.requests.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "request not found (the log keeps the last 500)")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// generatedImage renders seed product artwork: /media/generated/{product}/{n}.svg?c=Category&h=hue
func (s *server) generatedImage(w http.ResponseWriter, r *http.Request) {
	variant, err := strconv.Atoi(strings.TrimSuffix(r.PathValue("file"), ".svg"))
	hue, herr := strconv.Atoi(r.URL.Query().Get("h"))
	if err != nil || herr != nil || variant < 0 || variant >= media.ShotCount() || hue < -360 || hue > 720 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	_, _ = w.Write(media.ProductSVG(r.URL.Query().Get("c"), hue, variant))
}
