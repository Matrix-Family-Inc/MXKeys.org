/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"strings"
)

// withAdminAccess guards admin-only operational routes with a bearer
// token. The token is a plain shared secret set in
// security.admin_access_token. It is not a product tier: the routes
// are simply ops/debug surfaces that an operator does not want to
// expose anonymously.
func (s *Server) withAdminAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.adminAccessToken == "" {
			writeMatrixError(w, http.StatusNotFound, "M_NOT_FOUND", "Not found")
			return
		}
		token := adminTokenFromRequest(r)
		if !secureTokenCompare(token, s.adminAccessToken) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer realm="mxkeys-admin"`)
			writeMatrixError(w, http.StatusUnauthorized, "M_UNAUTHORIZED", "Admin access token required")
			return
		}
		next(w, r)
	}
}

// adminTokenFromRequest extracts the admin bearer token from either
// the standard Authorization header or the MXKeys-specific
// X-MXKeys-Admin-Token header.
func adminTokenFromRequest(r *http.Request) string {
	if authz := strings.TrimSpace(r.Header.Get("Authorization")); authz != "" {
		parts := strings.Fields(authz)
		if len(parts) >= 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(strings.Join(parts[1:], " "))
		}
	}
	return strings.TrimSpace(r.Header.Get("X-MXKeys-Admin-Token"))
}
