/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
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
	// Exposition errors (client disconnect mid-body) are not actionable
	// here: status has already been committed by Header() above. Explicit
	// discard quiets errcheck without changing behavior.
	_, _ = h.registry.WriteTo(w)
}
