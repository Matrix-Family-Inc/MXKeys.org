/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package server

import (
	"strings"

	"mxkeys/internal/zero/metrics"
)

// Request rejection reasons
const (
	RejectReasonInvalidJSON       = "invalid_json"
	RejectReasonEmptyRequest      = "empty_request"
	RejectReasonTooManyServers    = "too_many_servers"
	RejectReasonBodyTooLarge      = "body_too_large"
	RejectReasonMethodNotAllowed  = "method_not_allowed"
	RejectReasonInvalidServerName = "invalid_server_name"
	RejectReasonInvalidKeyID      = "invalid_key_id"
	RejectReasonMissingRequestID  = "missing_request_id"
	RejectReasonInvalidRequestID  = "invalid_request_id"
)

// Query status
const (
	QueryStatusSuccess = "success"
	QueryStatusFailure = "failure"
)

var (
	httpRequestsTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDurationSeconds = metrics.NewHistogramVec(
		metrics.HistogramOpts{
			Namespace: "mxkeys",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)

	inFlightRequests = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: "mxkeys",
			Name:      "in_flight_requests",
			Help:      "Current number of in-flight HTTP requests",
		},
	)

	keyQueriesTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "key_queries_total",
			Help:      "Total number of key queries",
		},
		[]string{"status"},
	)

	keyQueryServersRequested = metrics.NewHistogram(
		metrics.HistogramOpts{
			Namespace: "mxkeys",
			Name:      "key_query_servers_requested",
			Help:      "Number of servers requested per query",
			Buckets:   []float64{1, 2, 5, 10, 20, 50, 100},
		},
	)

	cacheHitsTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "cache_hits_total",
			Help:      "Total cache hits",
		},
		[]string{"cache_type"},
	)

	cacheMissesTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "cache_misses_total",
			Help:      "Total cache misses",
		},
		[]string{"cache_type"},
	)

	keyFetchesTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "key_fetches_total",
			Help:      "Total key fetch attempts",
		},
		[]string{"status", "source"},
	)

	keyFetchDurationSeconds = metrics.NewHistogramVec(
		metrics.HistogramOpts{
			Namespace: "mxkeys",
			Name:      "key_fetch_duration_seconds",
			Help:      "Key fetch duration in seconds",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		},
		[]string{"status", "source"},
	)

	rateLimitedRequestsTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "rate_limited_requests_total",
			Help:      "Total number of rate limited requests",
		},
		[]string{"limiter"},
	)

	cachedKeys = metrics.NewGaugeVec(
		metrics.GaugeOpts{
			Namespace: "mxkeys",
			Name:      "cached_keys",
			Help:      "Number of cached keys",
		},
		[]string{"cache_type"},
	)

	requestRejectionsTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "request_rejections_total",
			Help:      "Total rejected requests by reason",
		},
		[]string{"reason"},
	)

	upstreamFailuresTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "upstream_failures_total",
			Help:      "Total upstream failures by reason",
		},
		[]string{"reason"},
	)

	refetchesTotal = metrics.NewCounterVec(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "refetches_total",
			Help:      "Total refetch operations by reason",
		},
		[]string{"reason"},
	)

	negativeCacheHitsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "negative_cache_hits_total",
			Help:      "Total negative cache hits",
		},
	)

	negativeCacheWritesTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Name:      "negative_cache_writes_total",
			Help:      "Total negative cache writes",
		},
	)
)

func RecordHTTPRequest(method, route, status string, durationSeconds float64) {
	httpRequestsTotal.WithLabelValues(method, route, status).Inc()
	httpRequestDurationSeconds.WithLabelValues(method, route).Observe(durationSeconds)
}

func IncInFlightRequests() {
	inFlightRequests.Inc()
}

func DecInFlightRequests() {
	inFlightRequests.Dec()
}

func RecordKeyQuery(status string, serversRequested int) {
	keyQueriesTotal.WithLabelValues(status).Inc()
	keyQueryServersRequested.Observe(float64(serversRequested))
}

func RecordCacheHit(cacheType string) {
	cacheHitsTotal.WithLabelValues(cacheType).Inc()
}

func RecordCacheMiss(cacheType string) {
	cacheMissesTotal.WithLabelValues(cacheType).Inc()
}

func RecordKeyFetch(status, source string, durationSeconds float64) {
	keyFetchesTotal.WithLabelValues(status, source).Inc()
	keyFetchDurationSeconds.WithLabelValues(status, source).Observe(durationSeconds)
}

func RecordRateLimited(limiter string) {
	rateLimitedRequestsTotal.WithLabelValues(limiter).Inc()
}

func SetCachedKeys(cacheType string, count int) {
	cachedKeys.WithLabelValues(cacheType).Set(float64(count))
}

func RecordRequestRejection(reason string) {
	requestRejectionsTotal.WithLabelValues(reason).Inc()
}

func RecordUpstreamFailure(reason string) {
	upstreamFailuresTotal.WithLabelValues(reason).Inc()
}

func RecordRefetch(reason string) {
	refetchesTotal.WithLabelValues(reason).Inc()
}

func RecordNegativeCacheHit() {
	negativeCacheHitsTotal.Inc()
}

func RecordNegativeCacheWrite() {
	negativeCacheWritesTotal.Inc()
}

// NormalizeRoute converts URL path to route template for metrics
func NormalizeRoute(path string) string {
	switch {
	case path == "/_matrix/key/v2/query":
		return "/_matrix/key/v2/query"
	case strings.HasPrefix(path, "/_matrix/key/v2/server"):
		return "/_matrix/key/v2/server"
	case path == "/_matrix/federation/v1/version":
		return "/_matrix/federation/v1/version"
	case path == "/_mxkeys/health":
		return "/_mxkeys/health"
	case path == "/_mxkeys/live":
		return "/_mxkeys/live"
	case path == "/_mxkeys/ready":
		return "/_mxkeys/ready"
	case path == "/_mxkeys/metrics":
		return "/_mxkeys/metrics"
	case strings.HasPrefix(path, "/_mxkeys/"):
		return "/_mxkeys/operational"
	default:
		return "/other"
	}
}
