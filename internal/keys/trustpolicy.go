/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package keys

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
)

// TrustPolicy evaluates server keys against configured trust rules
type TrustPolicy struct {
	mu sync.RWMutex

	enabled                 bool
	denyList                []string
	allowList               []string
	denyPatterns            []string // compiled wildcard patterns
	allowPatterns           []string
	requireNotarySignatures int
	maxKeyAgeHours          int
	requireWellKnown        bool
	requireValidTLS         bool
	blockPrivateIPs         bool
}

// TrustPolicyConfig holds policy configuration
type TrustPolicyConfig struct {
	Enabled                 bool
	DenyList                []string
	AllowList               []string
	RequireNotarySignatures int
	MaxKeyAgeHours          int
	RequireWellKnown        bool
	RequireValidTLS         bool
	BlockPrivateIPs         bool
}

// PolicyViolation describes a trust policy violation
type PolicyViolation struct {
	Rule       string
	ServerName string
	Details    string
}

func (v *PolicyViolation) Error() string {
	return fmt.Sprintf("policy violation [%s] for %s: %s", v.Rule, v.ServerName, v.Details)
}

// NewTrustPolicy creates a new trust policy engine
func NewTrustPolicy(cfg TrustPolicyConfig) *TrustPolicy {
	tp := &TrustPolicy{
		enabled:                 cfg.Enabled,
		denyList:                cfg.DenyList,
		allowList:               cfg.AllowList,
		requireNotarySignatures: cfg.RequireNotarySignatures,
		maxKeyAgeHours:          cfg.MaxKeyAgeHours,
		requireWellKnown:        cfg.RequireWellKnown,
		requireValidTLS:         cfg.RequireValidTLS,
		blockPrivateIPs:         cfg.BlockPrivateIPs,
	}

	// Compile patterns
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

	if cfg.Enabled {
		log.Info("Trust policy engine initialized",
			"deny_list_count", len(cfg.DenyList),
			"allow_list_count", len(cfg.AllowList),
			"require_notary_signatures", cfg.RequireNotarySignatures,
			"max_key_age_hours", cfg.MaxKeyAgeHours,
		)
	}

	return tp
}

// CheckServer validates if a server is allowed by policy
func (tp *TrustPolicy) CheckServer(serverName string) *PolicyViolation {
	if !tp.enabled {
		return nil
	}

	tp.mu.RLock()
	defer tp.mu.RUnlock()

	// Check deny list first
	if tp.isDenied(serverName) {
		return &PolicyViolation{
			Rule:       "deny_list",
			ServerName: serverName,
			Details:    "server is on deny list",
		}
	}

	// Check allow list (if configured)
	if len(tp.allowList) > 0 && !tp.isAllowed(serverName) {
		return &PolicyViolation{
			Rule:       "allow_list",
			ServerName: serverName,
			Details:    "server is not on allow list",
		}
	}

	// Check private IP blocking
	if tp.blockPrivateIPs {
		if violation := tp.checkPrivateIP(serverName); violation != nil {
			return violation
		}
	}

	if tp.requireWellKnown {
		if violation := tp.checkRequireWellKnown(serverName); violation != nil {
			return violation
		}
	}

	if tp.requireValidTLS {
		if violation := tp.checkRequireValidTLS(serverName); violation != nil {
			return violation
		}
	}

	return nil
}

// CheckResponse validates a key response against policy
func (tp *TrustPolicy) CheckResponse(serverName string, resp *ServerKeysResponse) *PolicyViolation {
	if !tp.enabled {
		return nil
	}

	tp.mu.RLock()
	defer tp.mu.RUnlock()

	// Check key age
	if tp.maxKeyAgeHours > 0 {
		maxAge := time.Duration(tp.maxKeyAgeHours) * time.Hour
		validUntil := time.UnixMilli(resp.ValidUntilTS)
		keyAge := time.Until(validUntil)

		// Key should be valid for at least some time, but not too far in future
		if keyAge > maxAge {
			return &PolicyViolation{
				Rule:       "max_key_age",
				ServerName: serverName,
				Details:    fmt.Sprintf("key validity %v exceeds max %v", keyAge, maxAge),
			}
		}
	}

	// Check notary signatures requirement
	if tp.requireNotarySignatures > 0 {
		notaryCount := tp.countNotarySignatures(serverName, resp)
		if notaryCount < tp.requireNotarySignatures {
			return &PolicyViolation{
				Rule:       "require_notary_signatures",
				ServerName: serverName,
				Details:    fmt.Sprintf("has %d notary signatures, requires %d", notaryCount, tp.requireNotarySignatures),
			}
		}
	}

	return nil
}

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

// matchWildcard performs simple wildcard matching
// Supports: *.example.com, example.*, *.spam.*
func matchWildcard(pattern, s string) bool {
	if pattern == "*" {
		return true
	}

	// Split pattern by *
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}

	// Check prefix
	if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
		return false
	}

	// Check suffix
	lastPart := parts[len(parts)-1]
	if lastPart != "" && !strings.HasSuffix(s, lastPart) {
		return false
	}

	// Check middle parts
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

// isPrivateIP checks if IP is in private range
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for additional reserved ranges
	reserved := []string{
		"0.0.0.0/8",
		"100.64.0.0/10",   // Carrier-grade NAT
		"169.254.0.0/16",  // Link-local
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"224.0.0.0/4",     // Multicast
		"240.0.0.0/4",     // Reserved
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

// Reload updates policy configuration
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

	// Recompile patterns
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

// Stats returns policy statistics
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
