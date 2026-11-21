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
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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
	entries *boundedCache[*wellKnownEntry]
}

func newWellKnownCache() *wellKnownCache {
	return &wellKnownCache{entries: newBoundedCache(maxResolverCacheEntries, wellKnownTTL, func(e *wellKnownEntry) bool { return e.isError })}
}

func wellKnownTTL(entry *wellKnownEntry) time.Duration {
	if !entry.isError {
		return 24 * time.Hour
	}
	switch entry.errType {
	case errNotFound:
		return 1 * time.Hour
	case errInvalid:
		return 30 * time.Minute
	case errTemporary:
		return 2 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (c *wellKnownCache) get(hostname string) (*wellKnownEntry, bool) {
	return c.entries.get(hostname, time.Now())
}

func (c *wellKnownCache) set(hostname string, entry *wellKnownEntry) {
	c.entries.set(hostname, entry, entry.fetchedAt)
}

// resolveWellKnown fetches .well-known/matrix/server and returns delegated host:port.
func (r *Resolver) resolveWellKnown(ctx context.Context, hostname string) (*wellKnownEntry, error) {
	if entry, ok := r.cache.get(hostname); ok {
		if entry.isError {
			recordNegativeCacheHit(ResolverTypeWellKnown)
			return nil, fmt.Errorf("cached well-known error for %s", hostname)
		}
		recordWellKnownCacheHit()
		return entry, nil
	}
	recordWellKnownCacheMiss()

	url := "https://" + hostname + "/.well-known/matrix/server"
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

	delegatedHost, delegatedPort, _ := parseServerName(strings.TrimSpace(wk.Server))
	if !validDelegatedHost(delegatedHost) {
		r.cacheErrorWithType(hostname, errInvalid)
		return nil, fmt.Errorf("invalid m.server in well-known: %q", wk.Server)
	}
	entry := &wellKnownEntry{
		host:      delegatedHost,
		port:      delegatedPort,
		fetchedAt: time.Now(),
	}
	r.cache.set(hostname, entry)
	r.updateCacheMetrics()

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
// The certificate must be valid for the delegated hostname in every
// branch, and the Host header carries the delegated name.
func (r *Resolver) resolveDelegated(ctx context.Context, wk *wellKnownEntry, originalServerName string) (*ResolvedServer, error) {
	hostname := wk.host
	port := wk.port
	authority := hostname
	if port != 0 {
		authority = net.JoinHostPort(hostname, fmt.Sprint(port))
	}

	// 3.1: delegated IP literal. 3.2: delegated hostname with explicit port.
	if net.ParseIP(hostname) != nil || port != 0 {
		if port == 0 {
			port = 8448
		}
		return &ResolvedServer{Host: hostname, Port: port, ServerName: originalServerName, TLSName: hostname, Authority: authority}, nil
	}

	// 3.3 and 3.4: SRV records for the delegated hostname.
	if resolved, err := r.resolveSRV(ctx, hostname, originalServerName); err == nil {
		return resolved, nil
	}
	if resolved, err := r.resolveSRVLegacy(ctx, hostname, originalServerName); err == nil {
		return resolved, nil
	}

	// 3.5: delegated hostname with default port 8448.
	return &ResolvedServer{Host: hostname, Port: 8448, ServerName: originalServerName, TLSName: hostname, Authority: hostname}, nil
}
