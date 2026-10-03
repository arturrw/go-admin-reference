package httpapi

import (
	"math"
	"math/rand/v2"
	"net/http"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
)

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
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
	// Simulated storefront traffic — there is no real storefront behind
	// this reference project.
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
		RequestsPerSec: 1100 + rand.IntN(380),
		OnlineUsers:    290 + rand.IntN(50),
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

type requestStats struct {
	Total       int     `json:"total"`
	SuccessRate float64 `json:"successRate"`
	P95Ms       float64 `json:"p95Ms"`
	Client4xx   int     `json:"client4xx"`
	Server5xx   int     `json:"server5xx"`
}

// listRequests supports ?limit=, ?class=2|4|5 and ?q= (substring of method, path or status).
func (s *server) listRequests(w http.ResponseWriter, r *http.Request) {
	limit := min(max(queryInt(r, "limit", 120), 1), 500)
	class := r.URL.Query().Get("class")
	q := strings.ToLower(r.URL.Query().Get("q"))

	all := s.requests.Recent(500, nil)
	items := s.requests.Recent(limit, func(e reqlog.Entry) bool {
		status := strconv.Itoa(e.Status)
		if class != "" && !strings.HasPrefix(status, class) {
			return false
		}
		return q == "" || strings.Contains(strings.ToLower(e.Method+" "+e.Path+" "+status), q)
	})

	st := requestStats{Total: len(all)}
	durations := make([]float64, 0, len(all))
	ok := 0
	for _, e := range all {
		durations = append(durations, e.DurationMs)
		switch {
		case e.Status < 400:
			ok++
		case e.Status < 500:
			st.Client4xx++
		default:
			st.Server5xx++
		}
	}
	if len(all) > 0 {
		slices.Sort(durations)
		st.SuccessRate = math.Round(float64(ok)/float64(len(all))*1000) / 10
		st.P95Ms = durations[int(float64(len(durations)-1)*.95)]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "stats": st})
}
