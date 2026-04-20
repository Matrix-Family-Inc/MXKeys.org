/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 07 2026 UTC
 * Status: Created
 */

package keys

import (
	"net"
	"strings"

	"mxkeys/internal/zero/log"
)

// Reload updates policy configuration.
func (tp *TrustPolicy) Reload(cfg TrustPolicyConfig) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.enabled = cfg.Enabled
	tp.denyList = cfg.DenyList
	tp.allowList = cfg.AllowList
	tp.requireNotarySignatures = cfg.RequireNotarySignatures
	tp.maxKeyAgeHours = cfg.MaxKeyAgeHours
	tp.requireWellKnown = cfg.RequireWellKnown
	tp.requireValidTLS = cfg.RequireValidTLS
	tp.blockPrivateIPs = cfg.BlockPrivateIPs

	tp.denyPatterns = nil
	tp.allowPatterns = nil
	for _, entry := range cfg.DenyList {
		if strings.Contains(entry, "*") {
			tp.denyPatterns = append(tp.denyPatterns, entry)
		}
	}
	for _, entry := range cfg.AllowList {
		if strings.Contains(entry, "*") {
			tp.allowPatterns = append(tp.allowPatterns, entry)
		}
	}

	log.Info("Trust policy reloaded",
		"enabled", cfg.Enabled,
		"deny_list_count", len(cfg.DenyList),
		"allow_list_count", len(cfg.AllowList),
	)
}

// Stats returns policy statistics.
func (tp *TrustPolicy) Stats() map[string]interface{} {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return map[string]interface{}{
		"enabled":                   tp.enabled,
		"deny_list_count":           len(tp.denyList),
		"allow_list_count":          len(tp.allowList),
		"require_notary_signatures": tp.requireNotarySignatures,
		"max_key_age_hours":         tp.maxKeyAgeHours,
		"require_well_known":        tp.requireWellKnown,
		"require_valid_tls":         tp.requireValidTLS,
		"block_private_ips":         tp.blockPrivateIPs,
	}
}

// matchWildcard performs simple wildcard matching.
func matchWildcard(pattern, s string) bool {
	if pattern == "*" {
		return true
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
		return false
	}

	lastPart := parts[len(parts)-1]
	if lastPart != "" && !strings.HasSuffix(s, lastPart) {
		return false
	}

	remaining := s
	if parts[0] != "" {
		remaining = remaining[len(parts[0]):]
	}
	if lastPart != "" {
		remaining = remaining[:len(remaining)-len(lastPart)]
	}

	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		idx := strings.Index(remaining, parts[i])
		if idx == -1 {
			return false
		}
		remaining = remaining[idx+len(parts[i]):]
	}

	return true
}

// isPrivateIP checks if IP is in private range.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	reserved := []string{
		"0.0.0.0/8",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
	}

	for _, cidr := range reserved {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
