/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"mxkeys/internal/zero/metrics"
)

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Public health probes (required for k8s/orchestration)
	s.mux.HandleFunc("GET /_mxkeys/health", s.handleHealth)
	s.mux.HandleFunc("GET /_mxkeys/live", s.handleLiveness)
	s.mux.HandleFunc("GET /_mxkeys/ready", s.handleReadiness)

	// Protected operational endpoints (status and metrics expose internal details)
	s.mux.HandleFunc("GET /_mxkeys/status", s.withOperationalAccess(s.handleStatus))
	s.mux.Handle("GET /_mxkeys/metrics", s.withOperationalAccessHandler(metrics.Handler()))

	// Matrix Key Server API v2
	s.mux.HandleFunc("GET /_matrix/key/v2/server", s.handleServerKeys)
	s.mux.HandleFunc("GET /_matrix/key/v2/server/{keyID}", s.handleServerKeys)

	// POST /_matrix/key/v2/query - notary query (stricter rate limit)
	s.mux.HandleFunc("POST /_matrix/key/v2/query", s.withQueryRateLimit(s.handleKeyQuery))

	// GET /_mxkeys/server-info - optional enrichment endpoint.
	// Shares the query-rate-limit bucket so an anonymous client
	// cannot weaponise it as an amplified scanner.
	s.mux.HandleFunc("GET /_mxkeys/server-info", s.withQueryRateLimit(s.handleServerInfo))

	// Version endpoint
	s.mux.HandleFunc("GET /_matrix/federation/v1/version", s.handleVersion)

	// Signed tree head (public, verifiable)
	s.mux.HandleFunc("GET /_mxkeys/transparency/signed-head", s.handleSignedTreeHead)

	// Public key discovery (required for external STH verification)
	s.mux.HandleFunc("GET /_mxkeys/notary/key", s.handleNotaryPublicKey)

	// Protected operational API endpoints
	s.registerEnterpriseRoutes()
}

// withQueryRateLimit wraps a handler with query-specific rate limiting
func (s *Server) withQueryRateLimit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		v := s.rateLimiter.getVisitor(ip)

		if !v.queryLimiter.Allow() {
			RecordRateLimited("query")
			writeRateLimitError(w)
			return
		}

		h(w, r)
	}
}

// withOperationalAccess protects operational endpoints with enterprise token when configured
func (s *Server) withOperationalAccess(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.enterpriseAccessToken != "" {
			token := enterpriseTokenFromRequest(r)
			if !secureTokenCompare(token, s.enterpriseAccessToken) {
				w.Header().Set("Content-Type", "application/json")
				writeMatrixError(w, http.StatusUnauthorized, "M_UNAUTHORIZED", "Operational access token required")
				return
			}
		}
		h(w, r)
	}
}

// withOperationalAccessHandler protects http.Handler with enterprise token when configured
func (s *Server) withOperationalAccessHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.enterpriseAccessToken != "" {
			token := enterpriseTokenFromRequest(r)
			if !secureTokenCompare(token, s.enterpriseAccessToken) {
				w.Header().Set("Content-Type", "application/json")
				writeMatrixError(w, http.StatusUnauthorized, "M_UNAUTHORIZED", "Operational access token required")
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

// secureTokenCompare compares fixed-size digests to avoid leaking token length.
func secureTokenCompare(provided, expected string) bool {
	providedDigest := sha256.Sum256([]byte(provided))
	expectedDigest := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) == 1
}

// Handler returns the HTTP handler with all middleware applied
func (s *Server) Handler() http.Handler {
	// Chain middleware: request ID -> security headers -> logging -> rate limiting -> request validation -> routes
	handler := http.Handler(s.mux)
	handler = RequestIDRequirementMiddleware(s.config.Security.RequireRequestID, handler)
	handler = s.rateLimiter.Middleware(handler)
	handler = loggingMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = RequestIDMiddleware(handler)
	return handler
}
