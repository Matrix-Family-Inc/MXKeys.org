/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import "net/http"

// registerEnterpriseRoutes registers protected operational routes.
func (s *Server) registerEnterpriseRoutes() {
	if s.enterpriseAccessToken == "" {
		return
	}

	register := func(pattern string, handler http.HandlerFunc) {
		s.mux.HandleFunc(pattern, s.withEnterpriseAccess(handler))
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
