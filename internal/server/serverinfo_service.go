/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// Orchestrator for /_mxkeys/server-info. Runs DNS / reachability
// / WHOIS in parallel under a single request budget and
// folds their results (plus per-subtask errors) into the cached
// ServerInfoResponse shape.

package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"mxkeys/internal/config"
)

// ServerInfoService wires a ServerInfoCache and the config flags
// that gate WHOIS into a single Enrich() method used by the HTTP
// handler.
type ServerInfoService struct {
	cache        *ServerInfoCache
	whoisEnabled bool
	cacheTTL     time.Duration
	reqTimeout   time.Duration
}

// NewServerInfoService constructs a service bound to an already-
// prepared cache.
func NewServerInfoService(cfg config.ServerInfoConfig, cache *ServerInfoCache) *ServerInfoService {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &ServerInfoService{
		cache:        cache,
		whoisEnabled: cfg.WhoisEnabled,
		cacheTTL:     ttl,
		reqTimeout:   timeout,
	}
}

// Enrich returns a ServerInfoResponse for serverName, serving
// from cache when fresh and running a fan-out otherwise. Any
// sub-task error is surfaced as a short string under
// Errors[<stage>] rather than as a top-level failure; the
// handler still returns 200 OK because partial enrichment is
// often more useful than no enrichment at all.
func (s *ServerInfoService) Enrich(ctx context.Context, serverName string) (*ServerInfoResponse, error) {
	if s == nil {
		return nil, errors.New("server-info: service not configured")
	}
	if cached, err := s.cache.Get(ctx, serverName); err != nil {
		// cache I/O error is non-fatal: log via caller, compute fresh.
	} else if cached != nil {
		return cached, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, s.reqTimeout)
	defer cancel()

	resp := &ServerInfoResponse{
		ServerName: serverName,
		FetchedAt:  time.Now().UTC(),
	}

	var (
		mu   sync.Mutex
		errs = map[string]string{}
	)

	setErr := func(stage, msg string) {
		if msg == "" {
			return
		}
		mu.Lock()
		errs[stage] = msg
		mu.Unlock()
	}

	var wg sync.WaitGroup

	var dns *ServerInfoDNS
	var reach *ServerInfoReachability
	wg.Add(1)
	go func() {
		defer wg.Done()
		dns, reach = probeReachability(runCtx, serverName)
	}()

	var whois *ServerInfoWhois
	if s.whoisEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host, _ := splitHostPort(serverName)
			whois = runWhois(runCtx, host)
		}()
	}

	wg.Wait()

	if dns != nil && (len(dns.A) > 0 || len(dns.AAAA) > 0 || dns.WellKnownServer != "" || len(dns.SRV) > 0) {
		resp.DNS = dns
	} else if dns != nil && dns.ResolvedHost != "" {
		resp.DNS = dns
	}

	if reach != nil {
		resp.Reachability = reach
		if reach.Error != "" && !reach.Reachable {
			setErr("reachability", reach.Error)
		}
	}

	if whois != nil {
		resp.Whois = whois
	} else if s.whoisEnabled {
		setErr("whois", "no record returned")
	}

	if len(errs) > 0 {
		resp.Errors = errs
	}

	if resp.DNS != nil || resp.Reachability != nil || resp.Whois != nil {
		if err := s.cache.Put(ctx, serverName, resp, s.cacheTTL); err != nil {
			// cache write failure is non-fatal; the handler still returns
			// the fresh response to the caller.
			_ = err
		}
	}

	return resp, nil
}
