package profiling

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Lightweight per-frame CPU profiler for tick-level insights.

var (
	mu             sync.Mutex
	frameTotals    = make(map[string]time.Duration)
	rollingSamples []sample
	lastTopNCache  topNCache
)

type sample struct {
	t      time.Time
	totals map[string]time.Duration
}

type topNCache struct {
	expiresAt time.Time
	n         int
	value     string
}

// Track returns a stop function that records the elapsed time under the given name.
// Usage: defer profiling.Track("subsystem.Operation")()
func Track(name string) func() {
	start := time.Now()
	return func() {
		d := time.Since(start)
		mu.Lock()
		frameTotals[name] += d
		mu.Unlock()
	}
}

// ResetFrame clears current per-frame totals. Call at the start of each frame.
func ResetFrame() {
	now := time.Now()
	mu.Lock()
	// carry the just-finished frame totals into rolling window
	if len(frameTotals) > 0 {
		snapshot := make(map[string]time.Duration, len(frameTotals))
		for k, v := range frameTotals {
			snapshot[k] = v
		}
		rollingSamples = append(rollingSamples, sample{t: now, totals: snapshot})
	}
	// prune entries older than 1 second
	cutoff := now.Add(-1 * time.Second)
	if len(rollingSamples) > 0 {
		firstIdx := 0
		for firstIdx < len(rollingSamples) && rollingSamples[firstIdx].t.Before(cutoff) {
			firstIdx++
		}
		if firstIdx > 0 {
			rollingSamples = append([]sample(nil), rollingSamples[firstIdx:]...)
		}
	}
	for k := range frameTotals {
		delete(frameTotals, k)
	}
	mu.Unlock()
}

// Snapshot returns a copy of current per-frame totals.
func Snapshot() map[string]time.Duration {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]time.Duration, len(frameTotals))
	for k, v := range frameTotals {
		out[k] = v
	}
	return out
}

// Total returns the sum of all tracked durations this frame.
func Total() time.Duration {
	ss := Snapshot()
	var sum time.Duration
	for _, v := range ss {
		sum += v
	}
	return sum
}

// SumWithPrefix returns the sum of durations whose names start with any of the given prefixes.
func SumWithPrefix(prefixes ...string) time.Duration {
	ss := Snapshot()
	var sum time.Duration
	for k, v := range ss {
		for _, p := range prefixes {
			if strings.HasPrefix(k, p) {
				sum += v
				break
			}
		}
	}
	return sum
}

// Add adds an arbitrary duration under the given name to the current frame totals.
func Add(name string, d time.Duration) {
	if d <= 0 {
		return
	}
	mu.Lock()
	frameTotals[name] += d
	mu.Unlock()
}

// TopN formats top N durations from the current frame totals.
// Example: "renderer.Render:4.2ms, meshing.BuildGreedyMeshForChunk:2.1ms"
func TopN(n int) string {
	now := time.Now()
	mu.Lock()
	if lastTopNCache.value != "" && now.Before(lastTopNCache.expiresAt) && lastTopNCache.n == n {
		val := lastTopNCache.value
		mu.Unlock()
		return val
	}
	// aggregate over last 1 second window
	aggregated := make(map[string]time.Duration)
	cutoff := now.Add(-1 * time.Second)
	for _, s := range rollingSamples {
		if s.t.Before(cutoff) {
			continue
		}
		for k, v := range s.totals {
			aggregated[k] += v
		}
	}
	// include current in-progress frame
	for k, v := range frameTotals {
		aggregated[k] += v
	}
	mu.Unlock()

	type pair struct {
		name string
		dur  time.Duration
	}
	list := make([]pair, 0, len(aggregated))
	for k, v := range aggregated {
		list = append(list, pair{name: k, dur: v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].dur > list[j].dur })
	if n > len(list) {
		n = len(list)
	}
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ms := float64(list[i].dur.Microseconds()) / 1000.0
		parts = append(parts, list[i].name+":"+formatMs(ms))
	}
	result := strings.Join(parts, ", ")

	mu.Lock()
	lastTopNCache = topNCache{expiresAt: now.Add(1 * time.Second), n: n, value: result}
	mu.Unlock()
	return result
}

func formatMs(ms float64) string {
	// keep one decimal for readability
	return trimTrailingZerosF(ms) + "ms"
}

func trimTrailingZerosF(f float64) string {
	// Format with one decimal place; drop .0 if integer.
	// Avoid fmt to keep this tiny; manual logic is fine here.
	whole := int64(f)
	frac := int64((f-float64(whole))*10.0 + 0.0001)
	if frac <= 0 {
		return itoa(whole)
	}
	return itoa(whole) + "." + itoa(frac)
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	buf := make([]byte, 0, 20)
	for i > 0 {
		d := i % 10
		buf = append(buf, byte('0'+d))
		i /= 10
	}
	// reverse
	for l, r := 0, len(buf)-1; l < r; l, r = l+1, r-1 {
		buf[l], buf[r] = buf[r], buf[l]
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}
