package load

import (
	"maps"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// Latency is a distribution summary in milliseconds.
type Latency struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
	Max float64 `json:"max"`
}

// Point is one second of a run.
type Point struct {
	Second    int     `json:"second"`
	Offered   int64   `json:"offered"`
	Sent      int64   `json:"sent"`
	Completed int64   `json:"completed"`
	Dropped   int64   `json:"dropped"`
	Errors    int64   `json:"errors"`
	InFlight  int64   `json:"inFlight"`
	Latency   Latency `json:"latencyMs"`
}

// stats counts what a run did. Counters are atomic; the histograms, the
// status table and the per-second series share one mutex.
type stats struct {
	offered, sent, completed, dropped, errors, replays, inFlight atomic.Int64

	mu         sync.Mutex
	window     *hdrhistogram.Histogram // since the last point
	cumulative *hdrhistogram.Histogram
	statuses   map[string]int64
	series     []Point
	last       Point // totals at the last point, to take differences from
}

// Latencies from 1 µs to 60 s with three significant digits.
func newHistogram() *hdrhistogram.Histogram { return hdrhistogram.New(1, 60_000_000, 3) }

func newStats() *stats {
	return &stats{window: newHistogram(), cumulative: newHistogram(), statuses: map[string]int64{}}
}

func (s *stats) answered(status int, elapsed time.Duration) {
	s.completed.Add(1)
	micros := max(elapsed.Microseconds(), 1)
	s.mu.Lock()
	_ = s.window.RecordValue(micros)
	_ = s.cumulative.RecordValue(micros)
	s.statuses[strconv.Itoa(status)]++
	s.mu.Unlock()
}

func (s *stats) unanswered() {
	s.errors.Add(1)
	s.mu.Lock()
	s.statuses["no answer"]++
	s.mu.Unlock()
}

// point closes the current second: the difference of every counter since
// the last point, and the window's latency.
func (s *stats) point(second int) {
	totals := Point{
		Offered:   s.offered.Load(),
		Sent:      s.sent.Load(),
		Completed: s.completed.Load(),
		Dropped:   s.dropped.Load(),
		Errors:    s.errors.Load(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.series = append(s.series, Point{
		Second:    second,
		Offered:   totals.Offered - s.last.Offered,
		Sent:      totals.Sent - s.last.Sent,
		Completed: totals.Completed - s.last.Completed,
		Dropped:   totals.Dropped - s.last.Dropped,
		Errors:    totals.Errors - s.last.Errors,
		InFlight:  s.inFlight.Load(),
		Latency:   summarize(s.window),
	})
	s.last = totals
	s.window.Reset()
}

func summarize(h *hdrhistogram.Histogram) Latency {
	if h.TotalCount() == 0 {
		return Latency{}
	}
	ms := func(micros int64) float64 { return float64(micros) / 1000 }
	return Latency{
		P50: ms(h.ValueAtQuantile(50)),
		P95: ms(h.ValueAtQuantile(95)),
		P99: ms(h.ValueAtQuantile(99)),
		Max: ms(h.Max()),
	}
}

type totals struct {
	Offered, Sent, Completed, Dropped, Errors, Replays, InFlight int64
	Statuses                                                     map[string]int64
	Latency                                                      Latency
	Series                                                       []Point
}

func (s *stats) totals() totals {
	s.mu.Lock()
	defer s.mu.Unlock()
	return totals{
		Offered:   s.offered.Load(),
		Sent:      s.sent.Load(),
		Completed: s.completed.Load(),
		Dropped:   s.dropped.Load(),
		Errors:    s.errors.Load(),
		Replays:   s.replays.Load(),
		InFlight:  s.inFlight.Load(),
		Statuses:  maps.Clone(s.statuses),
		Latency:   summarize(s.cumulative),
		Series:    append([]Point(nil), s.series...),
	}
}
