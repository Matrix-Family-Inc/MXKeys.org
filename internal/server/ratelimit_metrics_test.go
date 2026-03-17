/*
Project: MXKeys - Matrix Federation Trust Infrastructure
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family
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
	beforeRateLimited := metricValueOrZero(t, beforeBody, `(?m)^mxkeys_rate_limited_requests_total ([0-9]+(?:\.[0-9]+)?)$`)

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
	afterRateLimited := metricValueOrZero(t, afterBody, `(?m)^mxkeys_rate_limited_requests_total ([0-9]+(?:\.[0-9]+)?)$`)
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
