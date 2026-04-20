/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package metrics

import (
	"sync"
	"sync/atomic"
)

// Histogram tracks observations in fixed buckets.
type Histogram struct {
	name    string
	help    string
	labels  []string
	buckets []float64
	values  sync.Map // map[string]*histogramData
}

// histogramData is the per-label-set bucket + sum + count state.
type histogramData struct {
	buckets []uint64
	sum     uint64 // stored as bits of float64
	count   uint64
	mu      sync.Mutex
}

// HistogramOpts are options for creating a histogram.
type HistogramOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
	Buckets   []float64
}

// DefaultBuckets are the default histogram buckets for request/operation
// durations measured in seconds.
var DefaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// HistogramValue is a histogram value bound to a specific label set.
type HistogramValue struct {
	h   *Histogram
	key string
}

// NewHistogram creates and registers a new histogram with the default registry.
func NewHistogram(opts HistogramOpts) *Histogram {
	return DefaultRegistry.NewHistogram(opts)
}

// NewHistogramVec creates and registers a new histogram with labels.
func NewHistogramVec(opts HistogramOpts, labels []string) *Histogram {
	return DefaultRegistry.NewHistogramVec(opts, labels)
}

// NewHistogram creates and registers a new histogram in this registry.
func (r *Registry) NewHistogram(opts HistogramOpts) *Histogram {
	return r.NewHistogramVec(opts, nil)
}

// NewHistogramVec creates and registers a new histogram with labels.
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

// Observe records a value with empty labels.
func (h *Histogram) Observe(v float64) {
	h.observe("", v)
}

// WithLabelValues returns a histogram value bound to the given label values.
func (h *Histogram) WithLabelValues(lvs ...string) *HistogramValue {
	key := labelKey(lvs)
	return &HistogramValue{h: h, key: key}
}

// Observe records a value on the labelled histogram.
func (hv *HistogramValue) Observe(v float64) {
	hv.h.observe(hv.key, v)
}

// observe increments the bucket for v, updates sum atomically, and bumps count.
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
