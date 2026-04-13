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
 * Status: Updated
 */

package keys

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	maxConcurrentFetches = 50
	defaultRetryAttempts = 3
	retryBackoffBase     = 200 * time.Millisecond
	maxFederationBody    = 1 << 20 // 1MB per upstream response
)

// TrustedNotaryKey holds a pinned notary key
type TrustedNotaryKey struct {
	ServerName string
	KeyID      string
	PublicKey  []byte
}

// Fetcher fetches keys from remote servers using proper server discovery.
type Fetcher struct {
	client          *http.Client
	resolver        *Resolver
	fallbackServers []string
	fetchSem        *semaphore.Weighted
	circuitBreaker  *CircuitBreaker
	trustedNotaries map[string]TrustedNotaryKey
	retryAttempts   int
	maxSignatures   int
	blockPrivateIPs atomic.Bool
}

// FetcherConfig holds fetcher configuration
type FetcherConfig struct {
	FallbackServers []string
	Timeout         time.Duration
	TrustedNotaries []TrustedNotaryKey
	RetryAttempts   int
	MaxSignatures   int
	BlockPrivateIPs *bool // nil = default (true), explicit false to disable SSRF protection
}

// NewFetcher creates a new remote key fetcher with server discovery support.
func NewFetcher(fallbackServers []string, timeout time.Duration) *Fetcher {
	return NewFetcherWithConfig(FetcherConfig{
		FallbackServers: fallbackServers,
		Timeout:         timeout,
		RetryAttempts:   defaultRetryAttempts,
	})
}

// NewFetcherWithConfig creates a new fetcher with full configuration.
func NewFetcherWithConfig(cfg FetcherConfig) *Fetcher {
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = defaultRetryAttempts
	}
	if cfg.MaxSignatures <= 0 {
		cfg.MaxSignatures = 10
	}

	trustedMap := make(map[string]TrustedNotaryKey)
	for _, tn := range cfg.TrustedNotaries {
		trustedMap[tn.ServerName] = tn
	}

	f := &Fetcher{
		client: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: false,
				},
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       20,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		resolver:        NewResolver(),
		fallbackServers: cfg.FallbackServers,
		fetchSem:        semaphore.NewWeighted(maxConcurrentFetches),
		circuitBreaker:  NewCircuitBreaker(5, 60*time.Second),
		trustedNotaries: trustedMap,
		retryAttempts:   cfg.RetryAttempts,
		maxSignatures:   cfg.MaxSignatures,
	}

	// SSRF protection: default to true for security
	blockPrivate := true
	if cfg.BlockPrivateIPs != nil {
		blockPrivate = *cfg.BlockPrivateIPs
	}
	f.blockPrivateIPs.Store(blockPrivate)

	return f
}
