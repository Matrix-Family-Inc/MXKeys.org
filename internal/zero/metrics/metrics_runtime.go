/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

package metrics

import "runtime"

type runtimeCollector struct {
	goroutines *Gauge
	threads    *Gauge
	heapAlloc  *Gauge
	heapSys    *Gauge
	heapInuse  *Gauge
}

var runtimeMetrics *runtimeCollector

func init() {
	runtimeMetrics = &runtimeCollector{
		goroutines: NewGauge(GaugeOpts{
			Namespace: "go",
			Name:      "goroutines",
			Help:      "Number of goroutines that currently exist",
		}),
		threads: NewGauge(GaugeOpts{
			Namespace: "go",
			Name:      "threads",
			Help:      "Number of OS threads created",
		}),
		heapAlloc: NewGauge(GaugeOpts{
			Namespace: "go",
			Subsystem: "memstats",
			Name:      "heap_alloc_bytes",
			Help:      "Number of heap bytes allocated and still in use",
		}),
		heapSys: NewGauge(GaugeOpts{
			Namespace: "go",
			Subsystem: "memstats",
			Name:      "heap_sys_bytes",
			Help:      "Number of heap bytes obtained from system",
		}),
		heapInuse: NewGauge(GaugeOpts{
			Namespace: "go",
			Subsystem: "memstats",
			Name:      "heap_inuse_bytes",
			Help:      "Number of heap bytes in use",
		}),
	}
}

func collectRuntimeMetrics() {
	runtimeMetrics.goroutines.Set(float64(runtime.NumGoroutine()))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	runtimeMetrics.heapAlloc.Set(float64(m.HeapAlloc))
	runtimeMetrics.heapSys.Set(float64(m.HeapSys))
	runtimeMetrics.heapInuse.Set(float64(m.HeapInuse))

	p := make([]runtime.StackRecord, 1)
	n, _ := runtime.ThreadCreateProfile(p)
	runtimeMetrics.threads.Set(float64(n))
}
