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
	"net/http"
	"strings"
)

func (s *Server) withEnterpriseAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.enterpriseAccessToken == "" {
			writeMatrixError(w, http.StatusNotFound, "M_NOT_FOUND", "Not found")
			return
		}
		token := enterpriseTokenFromRequest(r)
		if !secureTokenCompare(token, s.enterpriseAccessToken) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer realm="mxkeys-enterprise"`)
			writeMatrixError(w, http.StatusUnauthorized, "M_UNAUTHORIZED", "Enterprise access token required")
			return
		}
		next(w, r)
	}
}

func enterpriseTokenFromRequest(r *http.Request) string {
	if authz := strings.TrimSpace(r.Header.Get("Authorization")); authz != "" {
		parts := strings.Fields(authz)
		if len(parts) >= 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(strings.Join(parts[1:], " "))
		}
	}
	return strings.TrimSpace(r.Header.Get("X-MXKeys-Enterprise-Token"))
}
