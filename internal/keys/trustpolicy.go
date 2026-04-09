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
