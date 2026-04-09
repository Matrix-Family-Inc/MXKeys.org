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

package keys

import (
	"fmt"
	"net"
	"strings"
)

// isDenied checks if server matches deny list
func (tp *TrustPolicy) isDenied(serverName string) bool {
	// Exact match
	for _, denied := range tp.denyList {
		if !strings.Contains(denied, "*") && denied == serverName {
			return true
		}
	}

	// Pattern match
	for _, pattern := range tp.denyPatterns {
		if matchWildcard(pattern, serverName) {
			return true
		}
	}

	return false
}

// isAllowed checks if server matches allow list
func (tp *TrustPolicy) isAllowed(serverName string) bool {
	// Exact match
	for _, allowed := range tp.allowList {
		if !strings.Contains(allowed, "*") && allowed == serverName {
			return true
		}
	}

	// Pattern match
	for _, pattern := range tp.allowPatterns {
		if matchWildcard(pattern, serverName) {
			return true
		}
	}

	return false
}

// checkPrivateIP blocks requests to private IP ranges
func (tp *TrustPolicy) checkPrivateIP(serverName string) *PolicyViolation {
	// Extract hostname (without port)
	host := serverName
	if idx := strings.LastIndex(serverName, ":"); idx != -1 {
		// Check if it's not an IPv6 address
		if !strings.Contains(serverName, "[") {
			host = serverName[:idx]
		}
	}

	// Handle IPv6 brackets
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		host = host[1:strings.Index(host, "]")]
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Not an IP literal, allow
		return nil
	}

	if isPrivateIP(ip) {
		return &PolicyViolation{
			Rule:       "block_private_ips",
			ServerName: serverName,
			Details:    fmt.Sprintf("private IP address %s is blocked", ip),
		}
	}

	return nil
}

// countNotarySignatures counts signatures from servers other than origin
func (tp *TrustPolicy) countNotarySignatures(serverName string, resp *ServerKeysResponse) int {
	count := 0
	for signer := range resp.Signatures {
		if signer != serverName {
			count++
		}
	}
	return count
}

func (tp *TrustPolicy) checkRequireWellKnown(serverName string) *PolicyViolation {
	host, port, isIP := parseServerName(serverName)
	if host == "" {
		return &PolicyViolation{
			Rule:       "require_well_known",
			ServerName: serverName,
			Details:    "server name is empty",
		}
	}
	if isIP {
		return &PolicyViolation{
			Rule:       "require_well_known",
			ServerName: serverName,
			Details:    "IP literals bypass well-known delegation",
		}
	}
	if port != 0 {
		return &PolicyViolation{
			Rule:       "require_well_known",
			ServerName: serverName,
			Details:    "explicit port bypasses well-known delegation",
		}
	}
	return nil
}

func (tp *TrustPolicy) checkRequireValidTLS(serverName string) *PolicyViolation {
	host, _, isIP := parseServerName(serverName)
	if host == "" {
		return &PolicyViolation{
			Rule:       "require_valid_tls",
			ServerName: serverName,
			Details:    "server name is empty",
		}
	}
	if isIP {
		return &PolicyViolation{
			Rule:       "require_valid_tls",
			ServerName: serverName,
			Details:    "IP literals are not allowed when strict TLS validation is required",
		}
	}

	normalized := strings.ToLower(host)
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") || strings.HasSuffix(normalized, ".local") {
		return &PolicyViolation{
			Rule:       "require_valid_tls",
			ServerName: serverName,
			Details:    "local hostnames are not allowed when strict TLS validation is required",
		}
	}
	return nil
}
