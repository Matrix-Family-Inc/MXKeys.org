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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
			if err == nil {
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

// --- Well-known resolution ---

type wellKnownResponse struct {
	Server string `json:"m.server"`
}

type errorType int

const (
	errNone      errorType = iota
	errNotFound            // 404 - cache longer
	errTemporary           // network/timeout - cache shorter
	errInvalid             // malformed response - cache medium
)

type wellKnownEntry struct {
	host      string
	port      int
	fetchedAt time.Time
	isError   bool
	errType   errorType
}

type wellKnownCache struct {
	mu      sync.RWMutex
	entries map[string]*wellKnownEntry
}

func newWellKnownCache() *wellKnownCache {
	return &wellKnownCache{entries: make(map[string]*wellKnownEntry)}
}

func (c *wellKnownCache) get(hostname string) (*wellKnownEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[hostname]
	if !ok {
		return nil, false
	}

	var ttl time.Duration
	if !entry.isError {
		ttl = 24 * time.Hour
	} else {
		switch entry.errType {
		case errNotFound:
			ttl = 1 * time.Hour // 404 is likely permanent, cache longer
		case errInvalid:
			ttl = 30 * time.Minute // malformed response, medium cache
		case errTemporary:
			ttl = 2 * time.Minute // network/timeout, retry soon
		default:
			ttl = 5 * time.Minute
		}
	}

	if time.Since(entry.fetchedAt) > ttl {
		return nil, false
	}
	return entry, true
}

func (c *wellKnownCache) set(hostname string, entry *wellKnownEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[hostname] = entry
}

// resolveWellKnown fetches .well-known/matrix/server and returns delegated host:port.
func (r *Resolver) resolveWellKnown(ctx context.Context, hostname string) (*wellKnownEntry, error) {
	// Check cache
	if entry, ok := r.cache.get(hostname); ok {
		if entry.isError {
			recordNegativeCacheHit(ResolverTypeWellKnown)
			return nil, fmt.Errorf("cached well-known error for %s", hostname)
		}
		recordWellKnownCacheHit()
		return entry, nil
	}
	recordWellKnownCacheMiss()

	url := fmt.Sprintf("https://%s/.well-known/matrix/server", hostname)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		r.cacheErrorWithType(hostname, errTemporary)
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.cacheErrorWithType(hostname, errTemporary)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		r.cacheErrorWithType(hostname, errNotFound)
		return nil, fmt.Errorf("well-known not found (404)")
	}
	if resp.StatusCode != http.StatusOK {
		r.cacheErrorWithType(hostname, errTemporary)
		return nil, fmt.Errorf("well-known returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		r.cacheErrorWithType(hostname, errTemporary)
		return nil, err
	}

	var wk wellKnownResponse
	if err := json.Unmarshal(body, &wk); err != nil {
		r.cacheErrorWithType(hostname, errInvalid)
		return nil, err
	}

	if wk.Server == "" {
		r.cacheErrorWithType(hostname, errInvalid)
		return nil, fmt.Errorf("empty m.server in well-known")
	}

	delegatedHost, delegatedPort, _ := parseServerName(wk.Server)
	entry := &wellKnownEntry{
		host:      delegatedHost,
		port:      delegatedPort,
		fetchedAt: time.Now(),
	}
	r.cache.set(hostname, entry)

	log.Debug("Well-known resolved",
		"hostname", hostname,
		"delegated_host", delegatedHost,
		"delegated_port", delegatedPort,
	)

	return entry, nil
}

func (r *Resolver) cacheErrorWithType(hostname string, et errorType) {
	recordNegativeCacheWrite(ResolverTypeWellKnown)
	r.cache.set(hostname, &wellKnownEntry{
		fetchedAt: time.Now(),
		isError:   true,
		errType:   et,
	})
	r.updateCacheMetrics()
}

// resolveDelegated processes the delegated server name from well-known.
func (r *Resolver) resolveDelegated(ctx context.Context, wk *wellKnownEntry, originalServerName string) (*ResolvedServer, error) {
	hostname := wk.host
	port := wk.port
	isIP := net.ParseIP(hostname) != nil

	// 3.1: Delegated IP literal
	if isIP {
		if port == 0 {
			port = 8448
		}
		return &ResolvedServer{Host: hostname, Port: port, ServerName: originalServerName}, nil
	}

	// 3.2: Delegated hostname with explicit port
	if port != 0 {
		return &ResolvedServer{Host: hostname, Port: port, ServerName: originalServerName}, nil
	}

	// 3.3: SRV _matrix-fed._tcp for delegated hostname
	if resolved, err := r.resolveSRV(hostname, originalServerName); err == nil {
		return resolved, nil
	}

	// 3.4: SRV _matrix._tcp for delegated hostname (deprecated)
	if resolved, err := r.resolveSRVLegacy(hostname, originalServerName); err == nil {
		return resolved, nil
	}

	// 3.5: Delegated hostname with default port 8448
	return &ResolvedServer{Host: hostname, Port: 8448, ServerName: originalServerName}, nil
}

// --- SRV resolution ---

type srvEntry struct {
	target     string
	port       int
	fetchedAt  time.Time
	isError    bool
	isNotFound bool // NXDOMAIN or no records
}

type srvCache struct {
	mu      sync.RWMutex
	entries map[string]*srvEntry
}

func newSRVCache() *srvCache {
	return &srvCache{entries: make(map[string]*srvEntry)}
}

func (c *srvCache) get(key string) (*srvEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	var ttl time.Duration
	if !entry.isError {
		ttl = 1 * time.Hour
	} else if entry.isNotFound {
		ttl = 30 * time.Minute // NXDOMAIN - likely permanent
	} else {
		ttl = 2 * time.Minute // temporary DNS error
	}

	if time.Since(entry.fetchedAt) > ttl {
		return nil, false
	}
	return entry, true
}

func (c *srvCache) set(key string, entry *srvEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry
}

// resolveSRV looks up _matrix-fed._tcp SRV record.
func (r *Resolver) resolveSRV(hostname, serverName string) (*ResolvedServer, error) {
	return r.lookupSRV("_matrix-fed", "_tcp", hostname, serverName)
}

// resolveSRVLegacy looks up _matrix._tcp SRV record (deprecated).
func (r *Resolver) resolveSRVLegacy(hostname, serverName string) (*ResolvedServer, error) {
	return r.lookupSRV("_matrix", "_tcp", hostname, serverName)
}

func (r *Resolver) lookupSRV(service, proto, hostname, serverName string) (*ResolvedServer, error) {
	cacheKey := fmt.Sprintf("%s.%s.%s", service, proto, hostname)

	// Check cache
	if entry, ok := r.srvCache.get(cacheKey); ok {
		if entry.isError {
			recordNegativeCacheHit(ResolverTypeSRV)
			return nil, fmt.Errorf("cached SRV error for %s", cacheKey)
		}
		recordSRVCacheHit()
		return &ResolvedServer{Host: entry.target, Port: entry.port, ServerName: serverName}, nil
	}
	recordSRVCacheMiss()

	_, addrs, err := net.LookupSRV(service, proto, hostname)
	if err != nil || len(addrs) == 0 {
		isNotFound := len(addrs) == 0 || isNXDOMAIN(err)
		recordNegativeCacheWrite(ResolverTypeSRV)
		r.srvCache.set(cacheKey, &srvEntry{fetchedAt: time.Now(), isError: true, isNotFound: isNotFound})
		r.updateCacheMetrics()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no SRV records for %s", cacheKey)
	}

	target := strings.TrimSuffix(addrs[0].Target, ".")
	port := int(addrs[0].Port)

	r.srvCache.set(cacheKey, &srvEntry{
		target:    target,
		port:      port,
		fetchedAt: time.Now(),
	})

	log.Debug("SRV resolved",
		"service", service,
		"hostname", hostname,
		"target", target,
		"port", port,
	)

	return &ResolvedServer{Host: target, Port: port, ServerName: serverName}, nil
}

func isNXDOMAIN(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	if ok := errors.As(err, &dnsErr); ok {
		return dnsErr.IsNotFound
	}
	return false
}

// updateCacheMetrics updates the negative cache size metrics
func (r *Resolver) updateCacheMetrics() {
	r.cache.mu.RLock()
	wkNegCount := 0
	for _, e := range r.cache.entries {
		if e.isError {
			wkNegCount++
		}
	}
	r.cache.mu.RUnlock()

	r.srvCache.mu.RLock()
	srvNegCount := 0
	for _, e := range r.srvCache.entries {
		if e.isError {
			srvNegCount++
		}
	}
	r.srvCache.mu.RUnlock()

	updateNegativeCacheSize(ResolverTypeWellKnown, wkNegCount)
	updateNegativeCacheSize(ResolverTypeSRV, srvNegCount)
}
