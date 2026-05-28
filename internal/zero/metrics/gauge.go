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

// Gauge is a metric that can go up and down. Values are stored as fixed-point
// integers scaled by 1e9 for atomic access.
type Gauge struct {
	name   string
	help   string
	labels []string
	values sync.Map // map[string]*int64
}

// GaugeOpts are options for creating a gauge.
type GaugeOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

// GaugeValue is a single gauge value bound to a specific label set.
type GaugeValue struct {
	ptr *int64
}

// NewGauge creates and registers a new gauge with the default registry.
func NewGauge(opts GaugeOpts) *Gauge {
	return DefaultRegistry.NewGauge(opts)
}

// NewGaugeVec creates and registers a new gauge with labels.
func NewGaugeVec(opts GaugeOpts, labels []string) *Gauge {
	return DefaultRegistry.NewGaugeVec(opts, labels)
}

// NewGauge creates and registers a new gauge in this registry.
func (r *Registry) NewGauge(opts GaugeOpts) *Gauge {
	return r.NewGaugeVec(opts, nil)
}

// NewGaugeVec creates and registers a new gauge with labels in this registry.
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

// Set sets the gauge to the given value.
func (g *Gauge) Set(v float64) {
	val, _ := g.values.LoadOrStore("", new(int64))
	atomic.StoreInt64(val.(*int64), int64(v*1e9))
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
	g.Add(-1)
}

// Add adds the given value to the gauge.
func (g *Gauge) Add(v float64) {
	val, _ := g.values.LoadOrStore("", new(int64))
	atomic.AddInt64(val.(*int64), int64(v*1e9))
}

// WithLabelValues returns a gauge value bound to the given label values.
func (g *Gauge) WithLabelValues(lvs ...string) *GaugeValue {
	key := labelKey(lvs)
	val, _ := g.values.LoadOrStore(key, new(int64))
	return &GaugeValue{ptr: val.(*int64)}
}

// Set sets the labelled gauge value.
func (gv *GaugeValue) Set(v float64) {
	atomic.StoreInt64(gv.ptr, int64(v*1e9))
}

// Inc increments the labelled gauge by 1.
func (gv *GaugeValue) Inc() {
	atomic.AddInt64(gv.ptr, 1e9)
}

// Dec decrements the labelled gauge by 1.
func (gv *GaugeValue) Dec() {
	atomic.AddInt64(gv.ptr, -1e9)
}

// Add adds the given value to the labelled gauge.
func (gv *GaugeValue) Add(v float64) {
	atomic.AddInt64(gv.ptr, int64(v*1e9))
}
