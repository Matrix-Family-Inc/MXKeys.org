/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
)

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
