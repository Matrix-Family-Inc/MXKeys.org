/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package metrics

import (
	"net/http"
)

// Handler returns an http.Handler that serves metrics
func Handler() http.Handler {
	return &metricsHandler{registry: DefaultRegistry}
}

// HandlerFor returns an http.Handler for a specific registry
func HandlerFor(r *Registry) http.Handler {
	return &metricsHandler{registry: r}
}

type metricsHandler struct {
	registry *Registry
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collectRuntimeMetrics()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	h.registry.WriteTo(w)
}
