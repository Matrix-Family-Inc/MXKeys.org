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

// Counter is a monotonically increasing counter.
type Counter struct {
	name   string
	help   string
	labels []string
	values sync.Map // map[string]*uint64 (labelKey -> value)
}

// CounterOpts are options for creating a counter.
type CounterOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

// CounterValue is a single counter value bound to a specific label set.
type CounterValue struct {
	ptr *uint64
}

// NewCounter creates and registers a new counter with the default registry.
func NewCounter(opts CounterOpts) *Counter {
	return DefaultRegistry.NewCounter(opts)
}

// NewCounterVec creates and registers a new counter with labels.
func NewCounterVec(opts CounterOpts, labels []string) *Counter {
	return DefaultRegistry.NewCounterVec(opts, labels)
}

// NewCounter creates and registers a new counter in this registry.
func (r *Registry) NewCounter(opts CounterOpts) *Counter {
	return r.NewCounterVec(opts, nil)
}

// NewCounterVec creates and registers a new counter with labels in this registry.
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

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.Add(1)
}

// Add adds the given value to the counter.
func (c *Counter) Add(v uint64) {
	val, _ := c.values.LoadOrStore("", new(uint64))
	atomic.AddUint64(val.(*uint64), v)
}

// WithLabelValues returns a counter value bound to the given label values.
func (c *Counter) WithLabelValues(lvs ...string) *CounterValue {
	key := labelKey(lvs)
	val, _ := c.values.LoadOrStore(key, new(uint64))
	return &CounterValue{ptr: val.(*uint64)}
}

// Inc increments the labelled counter by 1.
func (cv *CounterValue) Inc() {
	atomic.AddUint64(cv.ptr, 1)
}

// Add adds the given value to the labelled counter.
func (cv *CounterValue) Add(v uint64) {
	atomic.AddUint64(cv.ptr, v)
}
