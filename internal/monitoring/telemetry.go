package monitoring

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxSampleWindow = 1000

type RouteStat struct {
	Count           uint64  `json:"count"`
	Errors          uint64  `json:"errors"`
	TotalDurationMs float64 `json:"total_duration_ms"`
}

type TelemetryTracker struct {
	mu           sync.RWMutex
	startTime    time.Time
	totalReqs    uint64
	success2xx   uint64
	clientErr4xx uint64
	serverErr5xx uint64
	durations    []float64
	durIndex     int
	routes       map[string]*RouteStat
}

var GlobalTelemetry = NewTelemetryTracker()

func NewTelemetryTracker() *TelemetryTracker {
	return &TelemetryTracker{
		startTime: time.Now().UTC(),
		durations: make([]float64, 0, maxSampleWindow),
		routes:    make(map[string]*RouteStat),
	}
}

func (t *TelemetryTracker) Record(route string, status int, d time.Duration) {
	ms := float64(d.Microseconds()) / 1000.0

	atomic.AddUint64(&t.totalReqs, 1)
	if status >= 200 && status < 300 {
		atomic.AddUint64(&t.success2xx, 1)
	} else if status >= 400 && status < 500 {
		atomic.AddUint64(&t.clientErr4xx, 1)
	} else if status >= 500 {
		atomic.AddUint64(&t.serverErr5xx, 1)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Rolling circular buffer
	if len(t.durations) < maxSampleWindow {
		t.durations = append(t.durations, ms)
	} else {
		t.durations[t.durIndex] = ms
		t.durIndex = (t.durIndex + 1) % maxSampleWindow
	}

	// Route tracking
	if route == "" {
		route = "unknown"
	}
	stat, exists := t.routes[route]
	if !exists {
		stat = &RouteStat{}
		t.routes[route] = stat
	}
	stat.Count++
	stat.TotalDurationMs += ms
	if status >= 400 {
		stat.Errors++
	}
}

type LatencyPercentiles struct {
	P50 float64 `json:"p50_ms"`
	P95 float64 `json:"p95_ms"`
	P99 float64 `json:"p99_ms"`
	Avg float64 `json:"avg_ms"`
	Max float64 `json:"max_ms"`
}

type DBPoolStats struct {
	TotalConns    int32 `json:"total_conns"`
	IdleConns     int32 `json:"idle_conns"`
	AcquiredConns int32 `json:"acquired_conns"`
	MaxConns      int32 `json:"max_conns"`
}

type SystemResources struct {
	AllocMB    float64 `json:"alloc_mb"`
	SysMB      float64 `json:"sys_mb"`
	Goroutines int     `json:"goroutines"`
	NumGC      uint32  `json:"num_gc"`
}

type RouteSummary struct {
	Route  string  `json:"route"`
	Count  uint64  `json:"count"`
	Errors uint64  `json:"errors"`
	AvgMs  float64 `json:"avg_ms"`
}

type TelemetrySnapshot struct {
	UptimeSeconds uint64             `json:"uptime_seconds"`
	TotalRequests uint64             `json:"total_requests"`
	Success2xx    uint64             `json:"success_2xx"`
	ClientErr4xx  uint64             `json:"client_err_4xx"`
	ServerErr5xx  uint64             `json:"server_err_5xx"`
	ErrorRatePct  float64            `json:"error_rate_pct"`
	CurrentRPS    float64            `json:"current_rps"`
	Latency       LatencyPercentiles `json:"latency"`
	Resources     SystemResources    `json:"resources"`
	Database      DBPoolStats        `json:"database"`
	TopRoutes     []RouteSummary     `json:"top_routes"`
}

func (t *TelemetryTracker) GetSnapshot(dbPool *pgxpool.Pool) TelemetrySnapshot {
	t.mu.RLock()
	durCopy := make([]float64, len(t.durations))
	copy(durCopy, t.durations)

	routesCopy := make([]RouteSummary, 0, len(t.routes))
	for r, s := range t.routes {
		avg := 0.0
		if s.Count > 0 {
			avg = s.TotalDurationMs / float64(s.Count)
		}
		routesCopy = append(routesCopy, RouteSummary{
			Route:  r,
			Count:  s.Count,
			Errors: s.Errors,
			AvgMs:  avg,
		})
	}
	t.mu.RUnlock()

	uptime := uint64(time.Since(t.startTime).Seconds())
	if uptime == 0 {
		uptime = 1
	}

	total := atomic.LoadUint64(&t.totalReqs)
	s5xx := atomic.LoadUint64(&t.serverErr5xx)
	s4xx := atomic.LoadUint64(&t.clientErr4xx)
	s2xx := atomic.LoadUint64(&t.success2xx)

	errRate := 0.0
	if total > 0 {
		errRate = (float64(s5xx) / float64(total)) * 100.0
	}

	rps := float64(total) / float64(uptime)

	// Calculate latency percentiles
	var lat LatencyPercentiles
	if len(durCopy) > 0 {
		sort.Float64s(durCopy)
		n := len(durCopy)
		lat.P50 = durCopy[int(float64(n)*0.50)]
		lat.P95 = durCopy[int(float64(n)*0.95)]
		p99Idx := int(float64(n) * 0.99)
		if p99Idx >= n {
			p99Idx = n - 1
		}
		lat.P99 = durCopy[p99Idx]
		lat.Max = durCopy[n-1]

		sum := 0.0
		for _, v := range durCopy {
			sum += v
		}
		lat.Avg = sum / float64(n)
	}

	// Runtime memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	res := SystemResources{
		AllocMB:    float64(m.Alloc) / (1024 * 1024),
		SysMB:      float64(m.Sys) / (1024 * 1024),
		Goroutines: runtime.NumGoroutine(),
		NumGC:      m.NumGC,
	}

	// Database pool
	var dbStats DBPoolStats
	if dbPool != nil {
		stat := dbPool.Stat()
		dbStats = DBPoolStats{
			TotalConns:    stat.TotalConns(),
			IdleConns:     stat.IdleConns(),
			AcquiredConns: stat.AcquiredConns(),
			MaxConns:      stat.MaxConns(),
		}
	}

	// Sort top routes by count desc
	sort.Slice(routesCopy, func(i, j int) bool {
		return routesCopy[i].Count > routesCopy[j].Count
	})
	if len(routesCopy) > 10 {
		routesCopy = routesCopy[:10]
	}

	return TelemetrySnapshot{
		UptimeSeconds: uptime,
		TotalRequests: total,
		Success2xx:    s2xx,
		ClientErr4xx:  s4xx,
		ServerErr5xx:  s5xx,
		ErrorRatePct:  errRate,
		CurrentRPS:    rps,
		Latency:       lat,
		Resources:     res,
		Database:      dbStats,
		TopRoutes:     routesCopy,
	}
}
