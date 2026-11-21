/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
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

	// blockPrivate reports whether .well-known requests may only reach
	// public addresses. Nil means always.
	blockPrivate func() bool
	lookupIP     func(ctx context.Context, network, host string) ([]net.IP, error)
	lookupSRV    func(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}

// NewResolver creates a new server name resolver.
func NewResolver() *Resolver {
	r := &Resolver{
		cache:     newWellKnownCache(),
		srvCache:  newSRVCache(),
		lookupIP:  net.DefaultResolver.LookupIP,
		lookupSRV: net.DefaultResolver.LookupSRV,
	}
	r.client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext:           r.dialWellKnown,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          50,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-HTTPS URL blocked")
			}
			return nil
		},
	}
	return r
}

// dialWellKnown connects only to public addresses while private-address
// blocking is on. The check runs on every dial, so redirects are covered.
func (r *Resolver) dialWellKnown(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if r.blockPrivate != nil && !r.blockPrivate() {
		return dialer.DialContext(ctx, network, address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := r.publicIPs(ctx, host)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// publicIPs resolves host and fails if any address is private.
func (r *Resolver) publicIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("private address %s is blocked", ip)
		}
		return []net.IP{ip}, nil
	}
	ips, err := r.lookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses for %s", host)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, fmt.Errorf("private address %s is blocked for %s", ip, host)
		}
	}
	return ips, nil
}

// ResolvedServer is where and how to reach a Matrix server for federation.
type ResolvedServer struct {
	Host       string // IP or hostname to connect to
	Port       int    // port to connect to
	ServerName string // original server name
	// TLSName is the name the certificate must be valid for: the host of
	// the server name, or the delegated hostname after .well-known. It
	// is never an SRV target.
	TLSName string
	// Authority is the Host header value the discovery step requires.
	Authority string
	PinnedIPs []string
}

// URL returns the base HTTPS URL for the resolved server.
func (r *ResolvedServer) URL() string {
	return "https://" + net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

// HostHeader returns the Host header to send.
func (r *ResolvedServer) HostHeader() string {
	if r.Authority != "" {
		return r.Authority
	}
	return r.ServerName
}

// TLSServerName returns the name used for SNI and certificate checks.
func (r *ResolvedServer) TLSServerName() string {
	if r.TLSName != "" {
		return r.TLSName
	}
	return r.Host
}

// ResolveServerName resolves a Matrix server name to a host:port
// following the Matrix server discovery algorithm.
func (r *Resolver) ResolveServerName(ctx context.Context, serverName string) (*ResolvedServer, error) {
	hostname, port, isIP := parseServerName(serverName)
	if hostname == "" {
		return nil, fmt.Errorf("empty server name")
	}

	// Step 1: IP literal. Step 2: hostname with explicit port.
	if isIP || port != 0 {
		if port == 0 {
			port = 8448
		}
		return &ResolvedServer{Host: hostname, Port: port, ServerName: serverName, TLSName: hostname, Authority: serverName}, nil
	}

	// Step 3: .well-known delegation.
	delegated, err := r.resolveWellKnown(ctx, hostname)
	if err == nil {
		return r.resolveDelegated(ctx, delegated, serverName)
	}
	log.Debug("Well-known lookup failed, trying SRV", "server", serverName, "error", err)

	// Steps 4 and 5: SRV records for the server name itself.
	if resolved, err := r.resolveSRV(ctx, hostname, serverName); err == nil {
		return resolved, nil
	}
	if resolved, err := r.resolveSRVLegacy(ctx, hostname, serverName); err == nil {
		return resolved, nil
	}

	// Step 6: hostname:8448.
	return &ResolvedServer{Host: hostname, Port: 8448, ServerName: serverName, TLSName: hostname, Authority: hostname}, nil
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

// validDelegatedHost rejects .well-known targets that are not plain
// hostnames or IP literals.
func validDelegatedHost(host string) bool {
	if host == "" || len(host) > 255 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, c := range host {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return false
		}
	}
	return !strings.HasPrefix(host, ".") && !strings.HasPrefix(host, "-")
}
