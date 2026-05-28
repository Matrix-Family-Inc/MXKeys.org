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
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
)

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
