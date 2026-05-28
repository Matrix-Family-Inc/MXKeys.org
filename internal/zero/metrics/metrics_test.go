package metrics

import (
	"strings"
	"sync"
	"testing"
)

func TestCounterAddAndLabels(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter(CounterOpts{Namespace: "ns", Name: "hits"})
	c.Inc()
	c.Add(5)

	v := c.WithLabelValues()
	v.Inc()

	if got := r.counters["ns_hits"]; got == nil {
		t.Fatal("counter must be registered under composed name")
	}
}

func TestCounterLabelValuesIsolatesSeries(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounterVec(CounterOpts{Namespace: "ns", Name: "labeled"}, []string{"env"})

	a := c.WithLabelValues("prod")
	b := c.WithLabelValues("dev")
	a.Add(10)
	b.Inc()

	// Sibling labels must not contaminate each other.
	if a.ptr == b.ptr {
		t.Fatal("distinct label sets must produce distinct counter cells")
	}
}

func TestGaugeOperations(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge(GaugeOpts{Namespace: "ns", Name: "inflight"})

	g.Set(5)
	g.Inc()
	g.Dec()
	g.Add(2.5)

	v := g.WithLabelValues()
	v.Set(10)
	v.Inc()
	v.Dec()
	v.Add(-0.5)
}

func TestHistogramObserveDefaultBuckets(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogram(HistogramOpts{Namespace: "ns", Name: "latency_seconds"})
	h.Observe(0.001)
	h.Observe(0.5)
	h.Observe(9)

	v := h.WithLabelValues()
	v.Observe(0.02)
}

func TestHistogramCustomBuckets(t *testing.T) {
	r := NewRegistry()
	h := r.NewHistogramVec(HistogramOpts{
		Namespace: "ns",
		Name:      "bytes",
		Buckets:   []float64{100, 1000, 10000},
	}, []string{"op"})

	h.WithLabelValues("read").Observe(50)
	h.WithLabelValues("read").Observe(500)
	h.WithLabelValues("write").Observe(5000)
}

func TestBuildName(t *testing.T) {
	tests := []struct {
		ns, sub, name, want string
	}{
		{"", "", "simple", "simple"},
		{"ns", "", "hits", "ns_hits"},
		{"", "sub", "hits", "sub_hits"},
		{"ns", "sub", "hits", "ns_sub_hits"},
	}
	for _, tc := range tests {
		got := buildName(tc.ns, tc.sub, tc.name)
		if got != tc.want {
			t.Errorf("buildName(%q,%q,%q) = %q, want %q", tc.ns, tc.sub, tc.name, got, tc.want)
		}
	}
}

func TestLabelKeyAndFormat(t *testing.T) {
	if got := labelKey(nil); got != "" {
		t.Errorf("empty labelKey = %q", got)
	}
	key := labelKey([]string{"a", "b"})
	if !strings.Contains(key, "\x00") {
		t.Errorf("expected NUL separator in labelKey, got %q", key)
	}

	names := []string{"env", "region"}
	out := formatLabels(names, key)
	if !strings.Contains(out, "env=") || !strings.Contains(out, "region=") {
		t.Errorf("formatLabels result missing names: %q", out)
	}
	if formatLabels(nil, "") != "" {
		t.Errorf("empty formatLabels must be empty")
	}
}

// TestCounterConcurrentAdd verifies that Add under concurrency is atomic:
// N goroutines each incrementing 1000 times must observe a final value of
// N*1000 in the label-free slot.
func TestCounterConcurrentAdd(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter(CounterOpts{Namespace: "ns", Name: "concurrent"})

	const workers = 8
	const iters = 1000

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				c.Add(1)
			}
		}()
	}
	wg.Wait()

	// Read the label-free slot directly.
	v, _ := c.values.Load("")
	ptr, ok := v.(*uint64)
	if !ok {
		t.Fatalf("counter storage shape unexpected: %T", v)
	}
	if *ptr != workers*iters {
		t.Fatalf("expected %d after concurrent Adds, got %d", workers*iters, *ptr)
	}
}
