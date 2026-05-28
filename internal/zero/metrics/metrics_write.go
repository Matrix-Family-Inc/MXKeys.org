/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

package metrics

import (
	"fmt"
	"io"
	"sort"
	"sync/atomic"
)

// WriteTo writes all metrics in Prometheus text format
func (r *Registry) WriteTo(w io.Writer) (int64, error) {
	var total int64

	r.mu.RLock()
	defer r.mu.RUnlock()

	var counterNames, gaugeNames, histoNames []string
	for name := range r.counters {
		counterNames = append(counterNames, name)
	}
	for name := range r.gauges {
		gaugeNames = append(gaugeNames, name)
	}
	for name := range r.histos {
		histoNames = append(histoNames, name)
	}
	sort.Strings(counterNames)
	sort.Strings(gaugeNames)
	sort.Strings(histoNames)

	for _, name := range counterNames {
		c := r.counters[name]
		n, err := writeCounter(w, c)
		total += n
		if err != nil {
			return total, err
		}
	}

	for _, name := range gaugeNames {
		g := r.gauges[name]
		n, err := writeGauge(w, g)
		total += n
		if err != nil {
			return total, err
		}
	}

	for _, name := range histoNames {
		h := r.histos[name]
		n, err := writeHistogram(w, h)
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func writeCounter(w io.Writer, c *Counter) (int64, error) {
	var total int64

	n, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", c.name, c.help, c.name)
	total += int64(n)
	if err != nil {
		return total, err
	}

	c.values.Range(func(key, value interface{}) bool {
		k := key.(string)
		v := atomic.LoadUint64(value.(*uint64))
		labels := formatLabels(c.labels, k)
		var line string
		if labels == "" {
			line = fmt.Sprintf("%s %d\n", c.name, v)
		} else {
			line = fmt.Sprintf("%s{%s} %d\n", c.name, labels, v)
		}
		n, err = io.WriteString(w, line)
		total += int64(n)
		return err == nil
	})

	return total, err
}

func writeGauge(w io.Writer, g *Gauge) (int64, error) {
	var total int64

	n, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", g.name, g.help, g.name)
	total += int64(n)
	if err != nil {
		return total, err
	}

	g.values.Range(func(key, value interface{}) bool {
		k := key.(string)
		v := float64(atomic.LoadInt64(value.(*int64))) / 1e9
		labels := formatLabels(g.labels, k)
		var line string
		if labels == "" {
			line = fmt.Sprintf("%s %g\n", g.name, v)
		} else {
			line = fmt.Sprintf("%s{%s} %g\n", g.name, labels, v)
		}
		n, err = io.WriteString(w, line)
		total += int64(n)
		return err == nil
	})

	return total, err
}

func writeHistogram(w io.Writer, h *Histogram) (int64, error) {
	var total int64

	n, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
	total += int64(n)
	if err != nil {
		return total, err
	}

	h.values.Range(func(key, value interface{}) bool {
		k := key.(string)
		data := value.(*histogramData)

		data.mu.Lock()
		bucketCounts := make([]uint64, len(data.buckets))
		copy(bucketCounts, data.buckets)
		sum := float64FromBits(atomic.LoadUint64(&data.sum))
		count := atomic.LoadUint64(&data.count)
		data.mu.Unlock()

		labels := formatLabels(h.labels, k)

		var cumulative uint64
		for i, bound := range h.buckets {
			cumulative += bucketCounts[i]
			var line string
			if labels == "" {
				line = fmt.Sprintf("%s_bucket{le=\"%g\"} %d\n", h.name, bound, cumulative)
			} else {
				line = fmt.Sprintf("%s_bucket{%s,le=\"%g\"} %d\n", h.name, labels, bound, cumulative)
			}
			n, err = io.WriteString(w, line)
			total += int64(n)
			if err != nil {
				return false
			}
		}

		var line string
		if labels == "" {
			line = fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", h.name, count)
		} else {
			line = fmt.Sprintf("%s_bucket{%s,le=\"+Inf\"} %d\n", h.name, labels, count)
		}
		n, err = io.WriteString(w, line)
		total += int64(n)
		if err != nil {
			return false
		}

		if labels == "" {
			line = fmt.Sprintf("%s_sum %g\n%s_count %d\n", h.name, sum, h.name, count)
		} else {
			line = fmt.Sprintf("%s_sum{%s} %g\n%s_count{%s} %d\n", h.name, labels, sum, h.name, labels, count)
		}
		n, err = io.WriteString(w, line)
		total += int64(n)
		return err == nil
	})

	return total, err
}
