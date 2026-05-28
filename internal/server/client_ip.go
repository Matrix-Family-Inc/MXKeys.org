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
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

var (
	clientIPPolicyMu      sync.RWMutex
	trustForwardedHeaders bool
	trustedProxyNetworks  []*net.IPNet
)

// ConfigureClientIPPolicy configures whether forwarded headers can influence client IP extraction.
// Forwarded headers are trusted only when the direct peer is a configured trusted proxy.
func ConfigureClientIPPolicy(trustForwarded bool, trustedProxies []string) error {
	networks := make([]*net.IPNet, 0, len(trustedProxies))
	for _, raw := range trustedProxies {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if ip := net.ParseIP(raw); ip != nil {
			maskBits := 32
			if ip.To4() == nil {
				maskBits = 128
			}
			networks = append(networks, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(maskBits, maskBits),
			})
			continue
		}
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy network %q: %w", raw, err)
		}
		networks = append(networks, network)
	}

	clientIPPolicyMu.Lock()
	trustForwardedHeaders = trustForwarded
	trustedProxyNetworks = networks
	clientIPPolicyMu.Unlock()
	return nil
}

func extractClientIP(r *http.Request) string {
	return clientIPFromRequest(r)
}

func extractIP(r *http.Request) string {
	return clientIPFromRequest(r)
}

func clientIPFromRequest(r *http.Request) string {
	directIP := directPeerIP(r)
	if shouldTrustForwardedHeaders(directIP) {
		if forwarded := forwardedClientIP(r); forwarded != nil {
			return forwarded.String()
		}
	}

	if directIP == nil {
		return r.RemoteAddr
	}
	return directIP.String()
}

func directPeerIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(strings.TrimSpace(r.RemoteAddr))
	}
	return net.ParseIP(strings.TrimSpace(host))
}

func forwardedClientIP(r *http.Request) net.IP {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		chain := forwardedIPChain(xff)
		for i := len(chain) - 1; i >= 0; i-- {
			if !isTrustedProxyIP(chain[i]) {
				return chain[i]
			}
		}
		if len(chain) > 0 {
			return chain[0]
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if parsed := net.ParseIP(xri); parsed != nil {
			return parsed
		}
	}
	return nil
}

func forwardedIPChain(header string) []net.IP {
	parts := strings.Split(header, ",")
	chain := make([]net.IP, 0, len(parts))
	for _, candidate := range parts {
		candidate = strings.TrimSpace(candidate)
		if parsed := net.ParseIP(candidate); parsed != nil {
			chain = append(chain, parsed)
		}
	}
	return chain
}

func isTrustedProxyIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	clientIPPolicyMu.RLock()
	defer clientIPPolicyMu.RUnlock()
	for _, network := range trustedProxyNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func shouldTrustForwardedHeaders(directIP net.IP) bool {
	clientIPPolicyMu.RLock()
	defer clientIPPolicyMu.RUnlock()

	if !trustForwardedHeaders || directIP == nil {
		return false
	}
	for _, network := range trustedProxyNetworks {
		if network.Contains(directIP) {
			return true
		}
	}
	return false
}
