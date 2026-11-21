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
	"errors"
	"fmt"
	"net"
	"strings"
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
	entries *boundedCache[*srvEntry]
}

func newSRVCache() *srvCache {
	return &srvCache{entries: newBoundedCache(maxResolverCacheEntries, srvTTL, func(e *srvEntry) bool { return e.isError })}
}

func srvTTL(entry *srvEntry) time.Duration {
	switch {
	case !entry.isError:
		return 1 * time.Hour
	case entry.isNotFound:
		return 30 * time.Minute
	default:
		return 2 * time.Minute
	}
}

func (c *srvCache) get(key string) (*srvEntry, bool) {
	return c.entries.get(key, time.Now())
}

func (c *srvCache) set(key string, entry *srvEntry) {
	c.entries.set(key, entry, entry.fetchedAt)
}

// resolveSRV looks up the _matrix-fed._tcp SRV record of hostname.
func (r *Resolver) resolveSRV(ctx context.Context, hostname, serverName string) (*ResolvedServer, error) {
	return r.lookupSRVRecord(ctx, "matrix-fed", hostname, serverName)
}

// resolveSRVLegacy looks up the deprecated _matrix._tcp SRV record.
func (r *Resolver) resolveSRVLegacy(ctx context.Context, hostname, serverName string) (*ResolvedServer, error) {
	return r.lookupSRVRecord(ctx, "matrix", hostname, serverName)
}

// lookupSRVRecord resolves an SRV record. The connection goes to the SRV
// target, but the certificate and the Host header stay bound to hostname,
// the name the SRV record was looked up for.
func (r *Resolver) lookupSRVRecord(ctx context.Context, service, hostname, serverName string) (*ResolvedServer, error) {
	cacheKey := fmt.Sprintf("_%s._tcp.%s", service, hostname)
	resolved := func(target string, port int) *ResolvedServer {
		return &ResolvedServer{Host: target, Port: port, ServerName: serverName, TLSName: hostname, Authority: hostname}
	}

	if entry, ok := r.srvCache.get(cacheKey); ok {
		if entry.isError {
			recordNegativeCacheHit(ResolverTypeSRV)
			return nil, fmt.Errorf("cached SRV error for %s", cacheKey)
		}
		recordSRVCacheHit()
		return resolved(entry.target, entry.port), nil
	}
	recordSRVCacheMiss()

	_, addrs, err := r.lookupSRV(ctx, service, "tcp", hostname)
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
	r.srvCache.set(cacheKey, &srvEntry{target: target, port: port, fetchedAt: time.Now()})
	r.updateCacheMetrics()

	log.Debug("SRV resolved", "service", service, "hostname", hostname, "target", target, "port", port)
	return resolved(target, port), nil
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

// updateCacheMetrics publishes the negative cache sizes.
func (r *Resolver) updateCacheMetrics() {
	_, wkNeg := r.cache.entries.stats()
	_, srvNeg := r.srvCache.entries.stats()
	updateNegativeCacheSize(ResolverTypeWellKnown, wkNeg)
	updateNegativeCacheSize(ResolverTypeSRV, srvNeg)
}
