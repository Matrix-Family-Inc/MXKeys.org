/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Thu 06 Feb 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mxkeys/internal/zero/log"
)

// Resolver resolves Matrix server names to host:port using the full
// Matrix server discovery algorithm (well-known, SRV, fallback).
type Resolver struct {
	client   *http.Client
	cache    *wellKnownCache
	srvCache *srvCache
}

// NewResolver creates a new server name resolver.
func NewResolver() *Resolver {
	return &Resolver{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				if req.URL.Scheme != "https" {
					return fmt.Errorf("redirect to non-HTTPS URL blocked")
				}
				host := req.URL.Hostname()
				if ip := net.ParseIP(host); ip != nil {
					if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
						return fmt.Errorf("redirect to private IP %s blocked", host)
					}
				}
				return nil
			},
		},
		cache:    newWellKnownCache(),
		srvCache: newSRVCache(),
	}
}

// ResolvedServer contains the result of server name resolution.
type ResolvedServer struct {
	Host       string // IP or hostname to connect to
	Port       int    // port to connect to
	ServerName string // original server name for Host header
	PinnedIPs  []string
}

// URL returns the base HTTPS URL for the resolved server.
func (r *ResolvedServer) URL() string {
	return "https://" + net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

// ResolveServerName resolves a Matrix server name to a host:port
// following the full Matrix server discovery algorithm.
func (r *Resolver) ResolveServerName(ctx context.Context, serverName string) (*ResolvedServer, error) {
	hostname, port, isIP := parseServerName(serverName)

	// Step 1: IP literal
	if isIP {
		if port == 0 {
			port = 8448
		}
		return &ResolvedServer{Host: hostname, Port: port, ServerName: serverName}, nil
	}

	// Step 2: Explicit port
	if port != 0 {
		return &ResolvedServer{Host: hostname, Port: port, ServerName: serverName}, nil
	}

	// Step 3: Try .well-known
	delegated, err := r.resolveWellKnown(ctx, hostname)
	if err == nil {
		return r.resolveDelegated(ctx, delegated, serverName)
	}

	log.Debug("Well-known lookup failed, trying SRV",
		"server", serverName,
		"error", err,
	)

	// Step 4: SRV _matrix-fed._tcp
	if resolved, err := r.resolveSRV(hostname, serverName); err == nil {
		return resolved, nil
	}

	// Step 5: SRV _matrix._tcp (deprecated)
	if resolved, err := r.resolveSRVLegacy(hostname, serverName); err == nil {
		return resolved, nil
	}

	// Step 6: Default fallback -- hostname:8448
	return &ResolvedServer{Host: hostname, Port: 8448, ServerName: serverName}, nil
}

// --- Server name parsing ---

// parseServerName extracts hostname, port and whether it's an IP literal.
func parseServerName(name string) (hostname string, port int, isIPLiteral bool) {
	name = strings.TrimSpace(name)

	// IPv6 literal: [::1] or [::1]:8448
	if strings.HasPrefix(name, "[") {
		closeBracket := strings.Index(name, "]")
		if closeBracket == -1 {
			return name, 0, false
		}
		hostname = name[1:closeBracket]
		rest := name[closeBracket+1:]
		if strings.HasPrefix(rest, ":") {
			p, err := strconv.Atoi(rest[1:])
			if err == nil && p > 0 && p <= 65535 {
				port = p
			}
		}
		return hostname, port, true
	}

	// Check for IPv4 with port: 1.2.3.4:8448
	// or hostname with port: matrix.org:443
	if colonIdx := strings.LastIndex(name, ":"); colonIdx != -1 {
		maybeHost := name[:colonIdx]
		maybePort := name[colonIdx+1:]
		if p, err := strconv.Atoi(maybePort); err == nil && p > 0 && p <= 65535 {
			hostname = maybeHost
			port = p
			isIPLiteral = net.ParseIP(hostname) != nil
			return
		}
	}

	// No port
	hostname = name
	isIPLiteral = net.ParseIP(hostname) != nil
	return
}
