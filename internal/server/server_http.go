/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"

	"mxkeys/internal/zero/metrics"
)

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Health checks, status and metrics
	s.mux.HandleFunc("GET /_mxkeys/health", s.handleHealth)
	s.mux.HandleFunc("GET /_mxkeys/live", s.handleLiveness)
	s.mux.HandleFunc("GET /_mxkeys/ready", s.handleReadiness)
	s.mux.HandleFunc("GET /_mxkeys/status", s.handleStatus)
	s.mux.Handle("GET /_mxkeys/metrics", metrics.Handler())

	// Matrix Key Server API v2
	// GET /_matrix/key/v2/server - own keys (no keyID)
	s.mux.HandleFunc("GET /_matrix/key/v2/server", s.handleServerKeys)
	// GET /_matrix/key/v2/server/{keyID} - own keys with keyID (Go 1.22+ path params)
	s.mux.HandleFunc("GET /_matrix/key/v2/server/{keyID}", s.handleServerKeys)

	// POST /_matrix/key/v2/query - notary query (stricter rate limit)
	s.mux.HandleFunc("POST /_matrix/key/v2/query", s.withQueryRateLimit(s.handleKeyQuery))

	// Version endpoint
	s.mux.HandleFunc("GET /_matrix/federation/v1/version", s.handleVersion)

	// Protected operational API endpoints
	s.registerEnterpriseRoutes()
}

// withQueryRateLimit wraps a handler with query-specific rate limiting
func (s *Server) withQueryRateLimit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		v := s.rateLimiter.getVisitor(ip)

		if !v.queryLimiter.Allow() {
			RecordRateLimited()
			writeRateLimitError(w)
			return
		}

		h(w, r)
	}
}

// Handler returns the HTTP handler with all middleware applied
func (s *Server) Handler() http.Handler {
	// Chain middleware: request ID -> security headers -> logging -> rate limiting -> routes
	handler := http.Handler(s.mux)
	handler = s.rateLimiter.Middleware(handler)
	handler = loggingMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = RequestIDRequirementMiddleware(s.config.Security.RequireRequestID, handler)
	handler = RequestIDMiddleware(handler)
	return handler
}
