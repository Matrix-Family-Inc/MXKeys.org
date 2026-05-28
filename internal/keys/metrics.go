/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package keys

import (
	"mxkeys/internal/zero/metrics"
)

// Label value constants for metrics
const (
	// Cache types
	CacheTypeMemory   = "memory"
	CacheTypeDatabase = "database"

	// Fetch status
	FetchStatusSuccess = "success"
	FetchStatusFailure = "failure"

	// Fetch source
	FetchSourceDirect  = "direct"
	FetchSourceRefetch = "refetch"

	// Refetch reasons
	RefetchReasonMinValidUntil = "minimum_valid_until_ts"
	RefetchReasonExpired       = "expired"
	RefetchReasonCacheMiss     = "cache_miss"

	// Upstream failure reasons
	UpstreamFailureTLS              = "tls_error"
	UpstreamFailureTimeout          = "timeout"
	UpstreamFailureHTTP             = "http_error"
	UpstreamFailureServerMismatch   = "server_name_mismatch"
	UpstreamFailureInvalidSignature = "invalid_signature"
	UpstreamFailureInvalidResponse  = "invalid_response"

	// Resolver cache types
	ResolverTypeWellKnown = "wellknown"
	ResolverTypeSRV       = "srv"
)

var (
	keyCacheHits = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "cache_hits_total",
			Help:      "Total cache hits by type",
		},
		[]string{"cache_type"},
	)

	keyCacheMisses = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "cache_misses_total",
			Help:      "Total cache misses by type",
		},
		[]string{"cache_type"},
	)

	keyFetchAttempts = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "fetch_attempts_total",
			Help:      "Total key fetch attempts",
		},
		[]string{"status", "source"},
	)

	keyFetchDuration = metrics.NewHistogramVec(
		metrics.HistogramOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "fetch_duration_seconds",
			Help:      "Key fetch duration in seconds",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		},
		[]string{"status", "source"},
	)

	memoryCacheSize = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "memory_cache_size",
			Help:      "Number of entries in positive memory cache",
		},
	)

	negativeCacheSize = metrics.NewGaugeVec(
		metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "resolver",
			Name:      "negative_cache_size",
			Help:      "Number of entries in negative cache",
		},
		[]string{"type"},
	)

	refetchesTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "refetches_total",
			Help:      "Total refetch operations by reason",
		},
		[]string{"reason"},
	)

	upstreamFailures = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "keys",
			Name:      "upstream_failures_total",
			Help:      "Upstream failures by reason",
		},
		[]string{"reason"},
	)

	resolverCacheHits = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "resolver",
			Name:      "cache_hits_total",
			Help:      "Resolver cache hits",
		},
		[]string{"type"},
	)

	resolverCacheMisses = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "resolver",
			Name:      "cache_misses_total",
			Help:      "Resolver cache misses",
		},
		[]string{"type"},
	)

	negativeCacheHits = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "resolver",
			Name:      "negative_cache_hits_total",
			Help:      "Negative cache hits by type",
		},
		[]string{"type"},
	)

	negativeCacheWrites = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "resolver",
			Name:      "negative_cache_writes_total",
			Help:      "Negative cache writes by type",
		},
		[]string{"type"},
	)
)

func recordMemoryCacheHit() {
	keyCacheHits.WithLabelValues("memory").Inc()
}

func recordMemoryCacheMiss() {
	keyCacheMisses.WithLabelValues("memory").Inc()
}

func recordDBCacheHit() {
	keyCacheHits.WithLabelValues("database").Inc()
}

func recordDBCacheMiss() {
	keyCacheMisses.WithLabelValues("database").Inc()
}

func recordFetchSuccess(source string, durationSeconds float64) {
	keyFetchAttempts.WithLabelValues(FetchStatusSuccess, source).Inc()
	keyFetchDuration.WithLabelValues(FetchStatusSuccess, source).Observe(durationSeconds)
}

func recordFetchFailure(source string, durationSeconds float64) {
	keyFetchAttempts.WithLabelValues(FetchStatusFailure, source).Inc()
	keyFetchDuration.WithLabelValues(FetchStatusFailure, source).Observe(durationSeconds)
}

func updateMemoryCacheSize(size int) {
	memoryCacheSize.Set(float64(size))
}

func recordRefetch(reason string) {
	refetchesTotal.WithLabelValues(reason).Inc()
}

func recordUpstreamFailure(reason string) {
	upstreamFailures.WithLabelValues(reason).Inc()
}

func recordWellKnownCacheHit() {
	resolverCacheHits.WithLabelValues("wellknown").Inc()
}

func recordWellKnownCacheMiss() {
	resolverCacheMisses.WithLabelValues("wellknown").Inc()
}

func recordSRVCacheHit() {
	resolverCacheHits.WithLabelValues("srv").Inc()
}

func recordSRVCacheMiss() {
	resolverCacheMisses.WithLabelValues("srv").Inc()
}

func recordNegativeCacheHit(cacheType string) {
	negativeCacheHits.WithLabelValues(cacheType).Inc()
}

func recordNegativeCacheWrite(cacheType string) {
	negativeCacheWrites.WithLabelValues(cacheType).Inc()
}

func updateNegativeCacheSize(cacheType string, size int) {
	negativeCacheSize.WithLabelValues(cacheType).Set(float64(size))
}
