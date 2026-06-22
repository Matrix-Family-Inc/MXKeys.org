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

package server

import "net/http"

// registerAdminRoutes wires admin-only operational routes. These
// routes are registered only when security.admin_access_token is
// configured; otherwise they are absent from the mux entirely.
// They cover ops/debug surfaces (transparency inspection,
// analytics, circuit-breaker state, cluster status, trust policy)
// that an operator runs locally and does not want to expose
// anonymously. They are not a product tier.
func (s *Server) registerAdminRoutes() {
	if s.adminAccessToken == "" {
		return
	}

	register := func(pattern string, handler http.HandlerFunc) {
		s.mux.HandleFunc(pattern, s.withAdminAccess(handler))
	}

	if s.transparency != nil {
		register("GET /_mxkeys/transparency/log", s.handleTransparencyLog)
		register("GET /_mxkeys/transparency/verify", s.handleTransparencyVerify)
		register("GET /_mxkeys/transparency/stats", s.handleTransparencyStats)
		register("GET /_mxkeys/transparency/proof", s.handleTransparencyProof)
	}

	register("GET /_mxkeys/analytics/summary", s.handleAnalyticsSummary)
	register("GET /_mxkeys/analytics/servers", s.handleAnalyticsServers)
	register("GET /_mxkeys/analytics/anomalies", s.handleAnalyticsAnomalies)
	register("GET /_mxkeys/analytics/rotators", s.handleAnalyticsTopRotators)

	register("GET /_mxkeys/circuits", s.handleCircuitBreakerStats)

	if s.cluster != nil {
		register("GET /_mxkeys/cluster/status", s.handleClusterStatus)
		register("GET /_mxkeys/cluster/nodes", s.handleClusterNodes)
	}

	if s.trustPolicy != nil {
		register("GET /_mxkeys/policy/status", s.handleTrustPolicyStatus)
		register("GET /_mxkeys/policy/check", s.handleTrustPolicyCheck)
	}
}
