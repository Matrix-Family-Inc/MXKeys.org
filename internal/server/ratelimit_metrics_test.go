/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Mar 16 2026 UTC
 * Status: Created
 */

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"mxkeys/internal/config"
	"mxkeys/internal/zero/metrics"
)

func metricValueOrZero(t *testing.T, body, pattern string) float64 {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) != 2 {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("failed to parse metric value %q: %v", m[1], err)
	}
	return v
}

func scrapeMetrics(t *testing.T) string {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/metrics", nil)
	metrics.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics endpoint returned %d", rr.Code)
	}
	return rr.Body.String()
}

func TestRateLimiterMiddlewareReturns429AndIncrementsCounter(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		GlobalRequestsPerSecond: 1,
		GlobalBurst:             1,
		QueryRequestsPerSecond:  1,
		QueryBurst:              1,
	})

	beforeBody := scrapeMetrics(t)
	beforeRateLimited := metricValueOrZero(t, beforeBody, `(?m)^mxkeys_rate_limited_requests_total\{limiter="global"\} ([0-9]+(?:\.[0-9]+)?)$`)

	okHandler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "203.0.113.10:12345"
	rr1 := httptest.NewRecorder()
	okHandler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request must pass, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "203.0.113.10:12346"
	rr2 := httptest.NewRecorder()
	okHandler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", rr2.Code)
	}
	if rr2.Header().Get("Retry-After") != "1" {
		t.Fatalf("expected Retry-After=1, got %q", rr2.Header().Get("Retry-After"))
	}

	var body map[string]string
	if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
		t.Fatalf("429 body must be valid JSON: %v", err)
	}
	if body["errcode"] != "M_LIMIT_EXCEEDED" {
		t.Fatalf("expected M_LIMIT_EXCEEDED, got %q", body["errcode"])
	}

	afterBody := scrapeMetrics(t)
	afterRateLimited := metricValueOrZero(t, afterBody, `(?m)^mxkeys_rate_limited_requests_total\{limiter="global"\} ([0-9]+(?:\.[0-9]+)?)$`)
	if afterRateLimited < beforeRateLimited+1 {
		t.Fatalf("expected rate limited counter to increase by >=1, before=%v after=%v", beforeRateLimited, afterRateLimited)
	}
}

func TestHandleKeyQueryRejectionReasonMetrics(t *testing.T) {
	s := &Server{
		config: &config.Config{},
	}

	beforeBody := scrapeMetrics(t)
	beforeInvalidJSON := metricValueOrZero(
		t,
		beforeBody,
		fmt.Sprintf(`(?m)^mxkeys_request_rejections_total\{reason=%q\} ([0-9]+(?:\.[0-9]+)?)$`, RejectReasonInvalidJSON),
	)
	beforeInvalidName := metricValueOrZero(
		t,
		beforeBody,
		fmt.Sprintf(`(?m)^mxkeys_request_rejections_total\{reason=%q\} ([0-9]+(?:\.[0-9]+)?)$`, RejectReasonInvalidServerName),
	)

	invalidJSONReq := httptest.NewRequest(http.MethodPost, "/_matrix/key/v2/query", strings.NewReader("{not-json}"))
	invalidJSONRec := httptest.NewRecorder()
	s.handleKeyQuery(invalidJSONRec, invalidJSONReq)
	if invalidJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON must return 400, got %d", invalidJSONRec.Code)
	}

	invalidNameReq := httptest.NewRequest(
		http.MethodPost,
		"/_matrix/key/v2/query",
		strings.NewReader(`{"server_keys":{"../etc/passwd":{"ed25519:k1":{}}}}`),
	)
	invalidNameRec := httptest.NewRecorder()
	s.handleKeyQuery(invalidNameRec, invalidNameReq)
	if invalidNameRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid server_name must return 400, got %d", invalidNameRec.Code)
	}

	afterBody := scrapeMetrics(t)
	afterInvalidJSON := metricValueOrZero(
		t,
		afterBody,
		fmt.Sprintf(`(?m)^mxkeys_request_rejections_total\{reason=%q\} ([0-9]+(?:\.[0-9]+)?)$`, RejectReasonInvalidJSON),
	)
	afterInvalidName := metricValueOrZero(
		t,
		afterBody,
		fmt.Sprintf(`(?m)^mxkeys_request_rejections_total\{reason=%q\} ([0-9]+(?:\.[0-9]+)?)$`, RejectReasonInvalidServerName),
	)

	if afterInvalidJSON < beforeInvalidJSON+1 {
		t.Fatalf("expected invalid_json rejection metric to increase, before=%v after=%v", beforeInvalidJSON, afterInvalidJSON)
	}
	if afterInvalidName < beforeInvalidName+1 {
		t.Fatalf("expected invalid_server_name rejection metric to increase, before=%v after=%v", beforeInvalidName, afterInvalidName)
	}
}

// TestRateLimiterLRUEvictsOldestOnCapacity verifies that when the visitor
// map is full, inserting a new visitor evicts exactly one LRU-tail entry and
// retains all more-recently-used entries.
func TestRateLimiterLRUEvictsOldestOnCapacity(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		GlobalRequestsPerSecond: 1,
		GlobalBurst:             1,
		QueryRequestsPerSecond:  1,
		QueryBurst:              1,
		MaxVisitors:             3,
	})
	defer rl.Stop()

	rl.getVisitor("10.0.0.1")
	rl.getVisitor("10.0.0.2")
	rl.getVisitor("10.0.0.3")

	if rl.order.Len() != 3 {
		t.Fatalf("expected 3 visitors, got %d", rl.order.Len())
	}

	// Access 10.0.0.1 again: it moves to front, 10.0.0.2 is now LRU tail.
	rl.getVisitor("10.0.0.1")

	// Insert a new visitor past capacity: 10.0.0.2 must be evicted.
	rl.getVisitor("10.0.0.4")

	if rl.order.Len() != 3 {
		t.Fatalf("expected capacity to hold at 3, got %d", rl.order.Len())
	}
	if _, ok := rl.visitors["10.0.0.2"]; ok {
		t.Fatal("LRU tail visitor should have been evicted")
	}
	for _, ip := range []string{"10.0.0.1", "10.0.0.3", "10.0.0.4"} {
		if _, ok := rl.visitors[ip]; !ok {
			t.Fatalf("expected %s to remain", ip)
		}
	}
}

// TestRateLimiterCleanupDropsIdleVisitors verifies the background cleanup
// predicate: visitors older than idleTTL are removed in LRU order.
func TestRateLimiterCleanupDropsIdleVisitors(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		GlobalRequestsPerSecond: 1,
		GlobalBurst:             1,
		QueryRequestsPerSecond:  1,
		QueryBurst:              1,
		MaxVisitors:             10,
		IdleTTL:                 50 * time.Millisecond,
	})
	defer rl.Stop()

	rl.getVisitor("198.51.100.1")
	rl.getVisitor("198.51.100.2")

	time.Sleep(80 * time.Millisecond)

	// Bump one visitor: it should survive cleanup.
	rl.getVisitor("198.51.100.2")

	rl.cleanup()

	if _, ok := rl.visitors["198.51.100.1"]; ok {
		t.Fatal("idle visitor should have been cleaned up")
	}
	if _, ok := rl.visitors["198.51.100.2"]; !ok {
		t.Fatal("recently-seen visitor must survive cleanup")
	}
}
