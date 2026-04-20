/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package metrics

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry holds all registered metrics
type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	histos   map[string]*Histogram
}

// DefaultRegistry is the global default registry
var DefaultRegistry = NewRegistry()

// NewRegistry creates a new metrics registry
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
		histos:   make(map[string]*Histogram),
	}
}

// Counter is a monotonically increasing counter
type Counter struct {
	name   string
	help   string
	labels []string
	values sync.Map // map[string]*uint64 (labelKey -> value)
}

// CounterOpts options for creating a counter
type CounterOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

// NewCounter creates and registers a new counter
func NewCounter(opts CounterOpts) *Counter {
	return DefaultRegistry.NewCounter(opts)
}

// NewCounterVec creates and registers a new counter with labels
func NewCounterVec(opts CounterOpts, labels []string) *Counter {
	return DefaultRegistry.NewCounterVec(opts, labels)
}

func (r *Registry) NewCounter(opts CounterOpts) *Counter {
	return r.NewCounterVec(opts, nil)
}

func (r *Registry) NewCounterVec(opts CounterOpts, labels []string) *Counter {
	name := buildName(opts.Namespace, opts.Subsystem, opts.Name)
	c := &Counter{
		name:   name,
		help:   opts.Help,
		labels: labels,
	}
	r.mu.Lock()
	r.counters[name] = c
	r.mu.Unlock()
	return c
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	c.Add(1)
}

// Add adds the given value to the counter
func (c *Counter) Add(v uint64) {
	key := ""
	val, _ := c.values.LoadOrStore(key, new(uint64))
	atomic.AddUint64(val.(*uint64), v)
}

// WithLabelValues returns a counter for the given label values
func (c *Counter) WithLabelValues(lvs ...string) *CounterValue {
	key := labelKey(lvs)
	val, _ := c.values.LoadOrStore(key, new(uint64))
	return &CounterValue{ptr: val.(*uint64)}
}

// CounterValue is a single counter value
type CounterValue struct {
	ptr *uint64
}

// Inc increments by 1
func (cv *CounterValue) Inc() {
	atomic.AddUint64(cv.ptr, 1)
}

// Add adds the given value
func (cv *CounterValue) Add(v uint64) {
	atomic.AddUint64(cv.ptr, v)
}

// Gauge is a metric that can go up and down
type Gauge struct {
	name   string
	help   string
	labels []string
	values sync.Map // map[string]*int64
}

// GaugeOpts options for creating a gauge
type GaugeOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

// NewGauge creates and registers a new gauge
func NewGauge(opts GaugeOpts) *Gauge {
	return DefaultRegistry.NewGauge(opts)
}

// NewGaugeVec creates and registers a new gauge with labels
func NewGaugeVec(opts GaugeOpts, labels []string) *Gauge {
	return DefaultRegistry.NewGaugeVec(opts, labels)
}

func (r *Registry) NewGauge(opts GaugeOpts) *Gauge {
	return r.NewGaugeVec(opts, nil)
}

func (r *Registry) NewGaugeVec(opts GaugeOpts, labels []string) *Gauge {
	name := buildName(opts.Namespace, opts.Subsystem, opts.Name)
	g := &Gauge{
		name:   name,
		help:   opts.Help,
		labels: labels,
	}
	r.mu.Lock()
	r.gauges[name] = g
	r.mu.Unlock()
	return g
}

// Set sets the gauge to the given value
func (g *Gauge) Set(v float64) {
	key := ""
	val, _ := g.values.LoadOrStore(key, new(int64))
	atomic.StoreInt64(val.(*int64), int64(v*1e9))
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(v float64) {
	key := ""
	val, _ := g.values.LoadOrStore(key, new(int64))
	atomic.AddInt64(val.(*int64), int64(v*1e9))
}

// WithLabelValues returns a gauge for the given label values
func (g *Gauge) WithLabelValues(lvs ...string) *GaugeValue {
	key := labelKey(lvs)
	val, _ := g.values.LoadOrStore(key, new(int64))
	return &GaugeValue{ptr: val.(*int64)}
}

// GaugeValue is a single gauge value
type GaugeValue struct {
	ptr *int64
}

// Set sets the gauge value
func (gv *GaugeValue) Set(v float64) {
	atomic.StoreInt64(gv.ptr, int64(v*1e9))
}

// Inc increments by 1
func (gv *GaugeValue) Inc() {
	atomic.AddInt64(gv.ptr, 1e9)
}

// Dec decrements by 1
func (gv *GaugeValue) Dec() {
	atomic.AddInt64(gv.ptr, -1e9)
}

// Add adds the given value
func (gv *GaugeValue) Add(v float64) {
	atomic.AddInt64(gv.ptr, int64(v*1e9))
}

// Histogram tracks observations in buckets
type Histogram struct {
	name    string
	help    string
	labels  []string
	buckets []float64
	values  sync.Map // map[string]*histogramData
}

type histogramData struct {
	buckets []uint64
	sum     uint64 // stored as bits of float64
	count   uint64
	mu      sync.Mutex
}

// HistogramOpts options for creating a histogram
type HistogramOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
	Buckets   []float64
}

// DefaultBuckets are the default histogram buckets
var DefaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// NewHistogram creates and registers a new histogram
func NewHistogram(opts HistogramOpts) *Histogram {
	return DefaultRegistry.NewHistogram(opts)
}

// NewHistogramVec creates and registers a new histogram with labels
func NewHistogramVec(opts HistogramOpts, labels []string) *Histogram {
	return DefaultRegistry.NewHistogramVec(opts, labels)
}

func (r *Registry) NewHistogram(opts HistogramOpts) *Histogram {
	return r.NewHistogramVec(opts, nil)
}

func (r *Registry) NewHistogramVec(opts HistogramOpts, labels []string) *Histogram {
	name := buildName(opts.Namespace, opts.Subsystem, opts.Name)
	buckets := opts.Buckets
	if len(buckets) == 0 {
		buckets = DefaultBuckets
	}
	h := &Histogram{
		name:    name,
		help:    opts.Help,
		labels:  labels,
		buckets: buckets,
	}
	r.mu.Lock()
	r.histos[name] = h
	r.mu.Unlock()
	return h
}

// Observe records a value
func (h *Histogram) Observe(v float64) {
	h.observe("", v)
}

// WithLabelValues returns a histogram for the given label values
func (h *Histogram) WithLabelValues(lvs ...string) *HistogramValue {
	key := labelKey(lvs)
	return &HistogramValue{h: h, key: key}
}

func (h *Histogram) observe(key string, v float64) {
	val, _ := h.values.LoadOrStore(key, &histogramData{
		buckets: make([]uint64, len(h.buckets)),
	})
	data := val.(*histogramData)

	data.mu.Lock()
	for i, bound := range h.buckets {
		if v <= bound {
			data.buckets[i]++
			break
		}
	}
	for {
		old := atomic.LoadUint64(&data.sum)
		newSum := float64FromBits(old) + v
		if atomic.CompareAndSwapUint64(&data.sum, old, float64ToBits(newSum)) {
			break
		}
	}
	atomic.AddUint64(&data.count, 1)
	data.mu.Unlock()
}

// HistogramValue is a single histogram value
type HistogramValue struct {
	h   *Histogram
	key string
}

// Observe records a value
func (hv *HistogramValue) Observe(v float64) {
	hv.h.observe(hv.key, v)
}

func buildName(namespace, subsystem, name string) string {
	if namespace != "" && subsystem != "" {
		return namespace + "_" + subsystem + "_" + name
	}
	if namespace != "" {
		return namespace + "_" + name
	}
	if subsystem != "" {
		return subsystem + "_" + name
	}
	return name
}

func labelKey(lvs []string) string {
	if len(lvs) == 0 {
		return ""
	}
	return strings.Join(lvs, "\x00")
}

func formatLabels(names []string, key string) string {
	if key == "" || len(names) == 0 {
		return ""
	}
	values := strings.Split(key, "\x00")
	if len(values) != len(names) {
		return key
	}
	var parts []string
	for i, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%q", name, values[i]))
	}
	return strings.Join(parts, ",")
}

func float64ToBits(f float64) uint64 {
	return math.Float64bits(f)
}

func float64FromBits(b uint64) float64 {
	return math.Float64frombits(b)
}
