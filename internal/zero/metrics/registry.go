/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package metrics

import "sync"

// Registry holds all registered metrics.
type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	histos   map[string]*Histogram
}

// DefaultRegistry is the global default registry used by top-level
// NewCounter/NewGauge/NewHistogram helpers.
var DefaultRegistry = NewRegistry()

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
		histos:   make(map[string]*Histogram),
	}
}
