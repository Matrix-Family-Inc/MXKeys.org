/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import "net/http"

// handleClusterStatus returns cluster status.
// GET /_mxkeys/cluster/status
func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.cluster == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	writeJSON(w, s.cluster.Stats())
}

// handleClusterNodes returns cluster nodes.
// GET /_mxkeys/cluster/nodes
func (s *Server) handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.cluster == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	nodes := s.cluster.Nodes()

	writeJSON(w, map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// handleTrustPolicyStatus returns trust policy status.
// GET /_mxkeys/policy/status
func (s *Server) handleTrustPolicyStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.trustPolicy == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	writeJSON(w, s.trustPolicy.Stats())
}

// handleTrustPolicyCheck checks a server against trust policy.
// GET /_mxkeys/policy/check?server=matrix.org
func (s *Server) handleTrustPolicyCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.trustPolicy == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	serverName := r.URL.Query().Get("server")
	if serverName == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"errcode": "M_INVALID_PARAM",
			"error":   "server parameter required",
		})
		return
	}

	violation := s.trustPolicy.CheckServer(serverName)
	if violation != nil {
		writeJSON(w, map[string]interface{}{
			"server":  serverName,
			"allowed": false,
			"violation": map[string]string{
				"rule":    violation.Rule,
				"details": violation.Details,
			},
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"server":  serverName,
		"allowed": true,
	})
}
