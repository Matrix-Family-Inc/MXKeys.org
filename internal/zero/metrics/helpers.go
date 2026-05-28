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
	"fmt"
	"math"
	"strings"
)

// buildName composes a Prometheus metric name from namespace, subsystem, and name parts.
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

// labelKey joins label values with a NUL separator (safe delimiter not allowed
// in Prometheus label values).
func labelKey(lvs []string) string {
	if len(lvs) == 0 {
		return ""
	}
	return strings.Join(lvs, "\x00")
}

// formatLabels renders a map of label names to values as Prometheus exposition
// "name=\"value\",..." syntax. Falls back to the raw key when shapes mismatch.
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
